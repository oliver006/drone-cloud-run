package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	Action string
	Dir    string

	// deployment service account token
	Token string

	// cloud run runtime info
	Runtime    string
	Project    string
	Region     string
	SvcAccount string

	// deployed service config
	ServiceName          string
	ImageName            string
	AllowUnauthenticated bool
	Concurrency          string
	Memory               string
	Timeout              string
	Environment          map[string]string
	EnvSecrets           []string

	AdditionalFlags map[string]string
}

const (
	TmpTokenFileLocation = "/tmp/token.json"
)

// using a var instead of const so tests can override this
var (
	GCloudCommand = "gcloud"
)

var (
	// populated by "go build"
	BuildDate string
	BuildHash string
	BuildTag  string
)

func getProjectFromToken(token string) string {
	data := struct {
		ProjectID string `json:"project_id"`
	}{}
	err := json.Unmarshal([]byte(token), &data)
	if err != nil {
		return ""
	}
	return data.ProjectID
}

func parseConfig() (*Config, error) {
	cfg := Config{
		Dir:        filepath.Join(os.Getenv("DRONE_WORKSPACE"), os.Getenv("PLUGIN_DIR")),
		Action:     os.Getenv("PLUGIN_ACTION"),
		Runtime:    os.Getenv("PLUGIN_RUNTIME"),
		Project:    os.Getenv("PLUGIN_PROJECT"),
		Region:     os.Getenv("PLUGIN_REGION"),
		SvcAccount: os.Getenv("PLUGIN_SVC_ACCOUNT"),
		Token:      os.Getenv("PLUGIN_TOKEN"),

		ServiceName:          os.Getenv("PLUGIN_SERVICE"),
		ImageName:            os.Getenv("PLUGIN_IMAGE"),
		AllowUnauthenticated: os.Getenv("PLUGIN_ALLOW_UNAUTHENTICATED") == "true",
		Concurrency:          os.Getenv("PLUGIN_CONCURRENCY"),
		Memory:               os.Getenv("PLUGIN_MEMORY"),
		Timeout:              os.Getenv("PLUGIN_TIMEOUT"),
	}

	envStr := os.Getenv("PLUGIN_ENVIRONMENT")
	if err := json.Unmarshal([]byte(envStr), &cfg.Environment); err != nil && envStr != "" {
		log.Printf("json.Unmarshal() err: %s", err)
		log.Printf("os.Getenv(PLUGIN_ENVIRONMENT): %s", envStr)
	}

	addlFlagsStr := os.Getenv("PLUGIN_ADDL_FLAGS")
	if err := json.Unmarshal([]byte(addlFlagsStr), &cfg.AdditionalFlags); err != nil && addlFlagsStr != "" {
		log.Printf("json.Unmarshal() err: %s", err)
		log.Printf("os.Getenv(PLUGIN_ADDL_FLAGS): %s", envStr)
		return nil, fmt.Errorf("failed to parse additional flags: [%s]", err)
	}

	PluginEnvSecretPrefix := "PLUGIN_ENV_SECRET_"
	for _, e := range os.Environ() {
		if s := strings.SplitN(e, "=", 2); len(s) > 0 && strings.HasPrefix(s[0], PluginEnvSecretPrefix) {
			k := strings.TrimPrefix(s[0], PluginEnvSecretPrefix)
			v := os.Getenv(s[0])
			cfg.EnvSecrets = append(cfg.EnvSecrets, fmt.Sprintf(`%s=%s`, k, v))
		}
	}

	if cfg.Action == "" {
		return nil, fmt.Errorf("Missing action")
	}
	if cfg.Runtime == "" {
		cfg.Runtime = "managed"
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("Missing service name")
	}
	if cfg.ImageName == "" {
		// for Drone v0.8 compat. as 'image' clashes since settings are passed top-level
		cfg.ImageName = os.Getenv("PLUGIN_DEPLOYMENT_IMAGE")
		if cfg.ImageName == "" {
			return nil, fmt.Errorf("Missing image/deployment_image name")
		}
	}

	if cfg.Token == "" {
		cfg.Token = os.Getenv("TOKEN")
		if cfg.Token == "" {
			return nil, fmt.Errorf("Missing token")
		}
	}

	if cfg.Project == "" {
		cfg.Project = getProjectFromToken(cfg.Token)
		if cfg.Project == "" {
			return nil, fmt.Errorf("project id not found in token or param")
		}
	}
	log.Printf("Using project ID: %s", cfg.Project)

	return &cfg, nil
}

func CreateExecutionPlan(cfg *Config) ([]string, error) {
	args := []string{
		"--quiet",
		"beta",
		"run",
	}
	switch cfg.Action {
	case "deploy":
		args = append(args, "deploy")
		args = append(args, cfg.ServiceName)
		args = append(args, "--image", cfg.ImageName)
		args = append(args, "--project", cfg.Project)
		args = append(args, "--platform", cfg.Runtime)

		if cfg.SvcAccount != "" {
			args = append(args, "--service-account", cfg.SvcAccount)
		}

		if len(cfg.EnvSecrets) > 0 || len(cfg.Environment) > 0 {
			e := make([]string, len(cfg.EnvSecrets))
			copy(e, cfg.EnvSecrets)
			for k, v := range cfg.Environment {
				e = append(e, fmt.Sprintf(`%s=%s`, k, v))
			}

			// we're using ":||:" as the separator for the args, let's hope no one puts that in an env variable value
			sep := ":||:"
			envStr := strings.Join(e, sep)
			envStr = "^" + sep + "^" + envStr
			args = append(args, "--set-env-vars", envStr)
		}

		if cfg.AllowUnauthenticated {
			args = append(args, "--allow-unauthenticated")
		}

		if cfg.Concurrency != "" {
			args = append(args, "--concurrency", cfg.Concurrency)
		}

		if cfg.Memory != "" {
			args = append(args, "--memory", cfg.Memory)
		}

		if cfg.Region != "" {
			args = append(args, "--region", cfg.Region)
		}

		for flg, argStr := range cfg.AdditionalFlags {
			if argStr != "" {
				args = append(args, fmt.Sprintf("--%s=%s", flg, argStr))
			} else {
				args = append(args, fmt.Sprintf("--%s", flg))
			}
		}

	default:
		return []string{}, fmt.Errorf("action: %s not implemented yet", cfg.Action)
	}

	return args, nil
}

func ExecutePlan(e *Env, plan []string) error {
	if err := e.Run(GCloudCommand, plan...); err != nil {
		return fmt.Errorf("error: %s\n", err)
	}

	return nil
}

func runConfig(cfg *Config) error {
	plan, err := CreateExecutionPlan(cfg)
	if err != nil {
		return err
	}

	e := NewEnv(cfg.Dir, os.Environ(), os.Stdout, os.Stderr, false)

	e.Run(GCloudCommand, "version")

	if err := e.Run(GCloudCommand, "auth", "activate-service-account", "--key-file", TmpTokenFileLocation); err != nil {
		return err
	}

	return ExecutePlan(e, plan)
}

type Env struct {
	dir    string
	env    []string
	stdout io.Writer
	stderr io.Writer
	dryRun bool
}

func NewEnv(dir string, env []string, stdout, stderr io.Writer, dryRun bool) *Env {
	return &Env{
		dir:    dir,
		env:    env,
		stdout: stdout,
		stderr: stderr,
		dryRun: dryRun,
	}
}

func (e *Env) Run(name string, arg ...string) error {
	log.Printf("Running: %s %#v", name, arg)
	if e.dryRun {
		return nil
	}
	cmd := exec.Command(name, arg...)
	cmd.Dir = e.dir
	cmd.Env = e.env
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr
	return cmd.Run()
}

func main() {
	if BuildTag == "" {
		BuildTag = "[not-tagged]"
	}
	log.Printf("drone-cloud-run plugin  version: %s   hash: %s   date: %s", BuildTag, BuildHash, BuildDate)

	showVersion := flag.Bool("v", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		os.Exit(0)
		return
	}

	cfg, err := parseConfig()
	if err != nil {
		log.Fatalf("parseConfig() err: %s", err)
		return
	}

	if err := ioutil.WriteFile(TmpTokenFileLocation, []byte(cfg.Token), 0600); err != nil {
		log.Fatalf("Error writing token file: %s", err)
	}

	defer func() {
		os.Remove(TmpTokenFileLocation)
	}()

	if err := runConfig(cfg); err != nil {
		log.Fatalf("runConfig() err: %s", err)
		return
	}
}
