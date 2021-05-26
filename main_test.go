package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

var (
	validGCPKey = `
{
  "type": "service_account",
  "project_id": "my-project-id",
  "private_key_id": "",
  "private_key": "",
  "client_email": "my-project@appspot.gserviceaccount.com",
  "client_id": "123",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/my-project%40appspot.gserviceaccount.com"
}
`

	invalidGCPKey = `
{
  "type": "service_account",
  234: "invalid Json    ,

}
`
)

func TestEnvironRun(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	e := NewEnv("/tmp", []string{"ABC=123"}, stdout, stderr, false)

	if err := e.Run("/bin/echo", "sup"); err == nil {
		if stdout.String() != "sup\n" {
			t.Errorf("got stdout : %s", stdout.String())
		}
		if stderr.String() != "" {
			t.Errorf("got stdout : %s", stderr.String())
		}
	} else {
		t.Errorf("got err: %s", err)
	}

	if err := e.Run("/usr/bin/env"); err == nil {
		if !strings.Contains(stdout.String(), "ABC=123") {
			t.Errorf("didn't find ABC in env, got: %s", stdout.String())
		}
	} else {
		t.Errorf("got err: %s", err)
	}

	stdout = &bytes.Buffer{}

	e.dryRun = true
	if err := e.Run("/bin/echo", "sup"); err == nil {
		if strings.Contains(stdout.String(), "sup") {
			t.Errorf("got stdout : %s", stdout.String())
		}
	} else {
		t.Errorf("got err: %s", err)
	}
}

func TestGetProjectFromToken(t *testing.T) {
	if id := getProjectFromToken(validGCPKey); id != "my-project-id" {
		t.Errorf("Wrong project id, got: %s", id)
	}

	if id := getProjectFromToken(invalidGCPKey); id != "" {
		t.Errorf("Expected empty id, got: %s", id)
	}
}

func TestParseAndRunConfig(t *testing.T) {
	for _, tst := range []struct {
		env                   map[string]string
		cfgExpectedOk         bool
		cfgExpectedProjectId  string
		cfgExpectedEnvSecrets []string
		cfgExpectedEnvKeys    []string
		cfgExpectedSecrets    map[string]string
		planExpectedOk        bool
		planExpectedFlags     []string
	}{
		{
			cfgExpectedOk:        true,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
			planExpectedOk:       true,
		},
		{
			cfgExpectedOk:         true,
			env:                   map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_ENV_SECRET_API_KEY": "secret", "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId:  "my-project-id",
			cfgExpectedEnvSecrets: []string{"API_KEY=secret"},
			planExpectedOk:        true,
		},
		{
			cfgExpectedOk:        true,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_ENVIRONMENT": `{"VAR_1":"var01","VERSION":"d0c13cb8646875cf94387f0d3de4e92b85eee3b0"}`, "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
			cfgExpectedEnvKeys:   []string{"VAR_1=var01", "VERSION=d0c13cb8646875cf94387f0d3de4e92b85eee3b0"},
			planExpectedOk:       true,
		},

		{
			cfgExpectedOk:        true,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_ENVIRONMENT": `    `, "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
			cfgExpectedEnvKeys:   []string{},
			planExpectedOk:       true,
		},

		// Test exposing secret as environment variable
		{
			cfgExpectedOk:        true,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_SECRETS": `{"var_1":"secretname:latest"}`, "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
			cfgExpectedSecrets:   map[string]string{"var_1": "secretname:latest"},
			planExpectedFlags:    []string{"--set-secrets", "^:||:^var_1=secretname:latest"},
			planExpectedOk:       true,
		},
		// Test exposing secret as file
		{
			cfgExpectedOk:        true,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_SECRETS": `{"var_1":"/mnt/path:secretname"}`, "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
			cfgExpectedSecrets:   map[string]string{"var_1": "/mnt/path:secretname"},
			planExpectedFlags:    []string{"--set-secrets", "^:||:^var_1=/mnt/path:secretname"},
			planExpectedOk:       true,
		},

		// test all the options to see if a proper execution plan is being created
		{
			cfgExpectedOk: true,
			env: map[string]string{
				"PLUGIN_ACTION":                "deploy",
				"PLUGIN_TOKEN":                 validGCPKey,
				"PLUGIN_SVC_ACCOUNT":           "1234-my-service-acct@account.com",
				"PLUGIN_SERVICE":               "my-service",
				"PLUGIN_IMAGE":                 "my-image",
				"PLUGIN_ALLOW_UNAUTHENTICATED": "true",
				"PLUGIN_CONCURRENCY":           "80",
				"PLUGIN_MEMORY":                "128Mi",
				"PLUGIN_TIMEOUT":               "10s",
				"PLUGIN_REGION":                "us-central1",
				"PLUGIN_ADDL_FLAGS":            `{"remove-cloudsql-instances":"my-proj:east2:db2"}`,
			},
			cfgExpectedProjectId: "my-project-id",
			planExpectedOk:       true,
			planExpectedFlags:    []string{"--remove-cloudsql-instances=my-proj:east2:db2"},
		},

		// parses ok but action is unknown
		{
			cfgExpectedOk:        true,
			env:                  map[string]string{"PLUGIN_ACTION": "unknown-action", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
			planExpectedOk:       false,
		},

		// use PLUGIN_DEPLOYMENT_IMAGE instead of PLUGIN_IMAGE, old drone :/
		{
			cfgExpectedOk:        true,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_SERVICE": "my-service", "PLUGIN_DEPLOYMENT_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
			planExpectedOk:       true,
		},

		// use TOKEN instead of PLUGIN_TOKEN, old drone :-/
		{
			cfgExpectedOk:        true,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "TOKEN": validGCPKey, "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
			planExpectedOk:       true,
		},

		// everything but TOKEN
		{
			cfgExpectedOk:        false,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
		},

		// everything but project
		{
			cfgExpectedOk:        false,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": "token", "PLUGIN_SERVICE": "my-service", "PLUGIN_DEPLOYMENT_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
		},
		{
			cfgExpectedOk:        false,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
		},
		{
			cfgExpectedOk:        false,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_SERVICE": "my-service"},
			cfgExpectedProjectId: "my-project-id",
		},
		{
			cfgExpectedOk:        false,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": "abcd", "PLUGIN_SERVICE": "my-service"},
			cfgExpectedProjectId: "my-project-id",
		},
		{
			cfgExpectedOk:        false,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "TOKEN": "", "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
		},
		{
			cfgExpectedOk:        false,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "TOKEN": "abcde", "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			cfgExpectedProjectId: "my-project-id",
		},
		{
			cfgExpectedOk:        false,
			env:                  map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service"},
			cfgExpectedProjectId: "my-project-id",
		},
		{
			cfgExpectedOk:        false,
			env:                  map[string]string{"PLUGIN_ACTION": "", "PLUGIN_SERVICE": "my-service"},
			cfgExpectedProjectId: "my-project-id",
		},
		{
			env: map[string]string{
				"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey,
				"PLUGIN_ADDL_FLAGS": `{"add-cloud-sql-instances":"instance1,instance2","clear-config-maps":""}`},
			cfgExpectedOk:        true,
			planExpectedOk:       true,
			cfgExpectedProjectId: "my-project-id",
			planExpectedFlags:    []string{"--add-cloud-sql-instances=instance1,instance2", "--clear-config-maps"},
		},
		{
			env: map[string]string{
				"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey,
				"PLUGIN_ADDL_FLAGS": `{"impossible-structure`},
			cfgExpectedOk:        false,
			cfgExpectedProjectId: "my-project-id",
		},
		{
			env: map[string]string{
				"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey,
				"PLUGIN_VARIANT": "alpha"},
			planExpectedOk:       true,
			cfgExpectedOk:        true,
			cfgExpectedProjectId: "my-project-id",
		},
		{
			env: map[string]string{
				"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey,
				"PLUGIN_VARIANT": "beta"},
			planExpectedOk:       true,
			cfgExpectedOk:        true,
			cfgExpectedProjectId: "my-project-id",
		},
		{
			env: map[string]string{
				"PLUGIN_ACTION": "update-traffic", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey,
				"PLUGIN_ADDL_FLAGS": `{"to-latest":""}`},
			planExpectedOk:       true,
			cfgExpectedOk:        true,
			cfgExpectedProjectId: "my-project-id",
			planExpectedFlags:    []string{"my-service"},
		},
		{
			env: map[string]string{
				"PLUGIN_ACTION": "update-traffic", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey,
				"PLUGIN_ADDL_FLAGS": `{"to-latest":""}`},
			planExpectedOk:       true,
			cfgExpectedOk:        true,
			cfgExpectedProjectId: "my-project-id",
			planExpectedFlags:    []string{"services", "update-traffic", "--to-latest"},
		},
		{
			env: map[string]string{
				"PLUGIN_ACTION": "update-traffic", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey,
				"PLUGIN_ADDL_FLAGS": `{"to-tags":"tag=100"}`},
			planExpectedOk:       true,
			cfgExpectedOk:        true,
			cfgExpectedProjectId: "my-project-id",
			planExpectedFlags:    []string{"--to-tags=tag=100"},
		},
		{
			env: map[string]string{
				"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey,
				"PLUGIN_ALLOW_UNAUTHENTICATED": "true"},
			planExpectedOk:       true,
			cfgExpectedOk:        true,
			cfgExpectedProjectId: "my-project-id",
			planExpectedFlags:    []string{"--allow-unauthenticated"},
		},
		{
			env: map[string]string{
				"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey,
				"PLUGIN_ALLOW_UNAUTHENTICATED": "false"},
			planExpectedOk:       true,
			cfgExpectedOk:        true,
			cfgExpectedProjectId: "my-project-id",
			planExpectedFlags:    []string{"--no-allow-unauthenticated"},
		},
		// gcloud defaults to --no-allow-unauthenticated if parameter not passed
		{
			env: map[string]string{
				"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service",
				"PLUGIN_IMAGE": "my-image", "PLUGIN_TOKEN": validGCPKey},
			planExpectedOk:       true,
			cfgExpectedOk:        true,
			cfgExpectedProjectId: "my-project-id",
			planExpectedFlags:    []string{"--no-allow-unauthenticated"},
		},
	} {
		name := fmt.Sprintf("env:[%s]", tst.env)
		t.Run(name, func(t *testing.T) {

			os.Clearenv()

			for k, v := range tst.env {
				os.Setenv(k, v)
			}

			cfg, err := parseConfig()
			if err != nil && tst.cfgExpectedOk == true {
				t.Errorf("parseConfig(  %#v  ) failed, err: %s", tst, err)
				return
			}
			if err == nil && tst.cfgExpectedOk == false {
				t.Errorf("parseConfig(  %#v  ) should have failed", tst)
				return
			}
			if !tst.cfgExpectedOk {
				return
			}

			if cfg.Project != tst.cfgExpectedProjectId {
				t.Errorf("expected projectID: %s   got: %s", tst.cfgExpectedProjectId, cfg.Project)
			}

			if cfg.Token == "" {
				t.Errorf("expected a token, got nothing, tst: %#v", tst)
			}

			for _, e := range tst.cfgExpectedEnvSecrets {
				found := false
				for _, s := range cfg.EnvSecrets {
					if s == e {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing env secret: %s, got: %#v", e, cfg.EnvSecrets)
				}
			}

			// for _, e := range tst.cfgExpectedSecrets {
			for k, v := range tst.cfgExpectedSecrets {
				found := v == cfg.Secrets[k]
				if !found {
					t.Errorf("missing secret: %s=%s, got: %#v", k, v, cfg.Secrets)
				}
			}

			if !tst.cfgExpectedOk {
				return
			}

			plan, err := CreateExecutionPlan(cfg)
			if err != nil && tst.planExpectedOk {
				t.Fatalf("plan was expected to be ok, got err: %s", err)
			} else if err == nil && !tst.planExpectedOk {
				t.Fatalf("Expected plan to fail, got plan: %v   env: %#v", plan, tst.env)
			}
			t.Logf("plan: %v", plan)

			if cfg.Variant == "alpha" && plan[1] != "alpha" {
				t.Fatal("execution plan should contain \"alpha\" for variant=alpha")
			}

			if cfg.Variant == "beta" && plan[1] != "beta" {
				t.Fatal("execution plan should contain \"beta\" for variant=beta")
			}

			if cfg.Variant == "" && len(plan) > 0 && plan[1] != "run" { // len(plan) is 0 for "gcloud version" command test
				t.Fatal("execution plan shouldn't contain any variant for variant=<empty string>")
			}

			for _, flg := range tst.planExpectedFlags {
				found := false
				for _, pflg := range plan {
					if pflg == flg {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("couldn't find expected flag [%s] in [%v]", flg, plan)
				}
			}

			GCloudCommand = "/bin/echo"
			err = runConfig(cfg)
			if err != nil && tst.planExpectedOk {
				t.Fatalf("plan was expected to be ok, got err: %s", err)
			} else if err == nil && !tst.planExpectedOk {
				t.Fatalf("Expected plan to fail, got plan: %v   env: %#v", plan, tst.env)
			}
		})
	}
}
