package main

import (
	"bytes"
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

func TestNope(t *testing.T) {
}

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
		if strings.Index(stdout.String(), "ABC=123") == -1 {
			t.Errorf("didn't find ABC in Env, got: %s", stdout.String())
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

func TestParseConfig(t *testing.T) {
	for _, tst := range []struct {
		Env                map[string]string
		expectedToBeOk     bool
		expectedProjectId  string
		expectedEnvSecrets []string
	}{
		{
			expectedToBeOk:    true,
			Env:               map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			expectedProjectId: "my-project-id",
		},
		{
			expectedToBeOk:     true,
			Env:                map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_ENV_SECRET_API_KEY": "secret", "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			expectedProjectId:  "my-project-id",
			expectedEnvSecrets: []string{"API_KEY=secret"},
		},

		// use PLUGIN_DEPLOYMENT_IMAGE instead of PLUGIN_IMAGE, old drone :/
		{
			expectedToBeOk:    true,
			Env:               map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_SERVICE": "my-service", "PLUGIN_DEPLOYMENT_IMAGE": "my-image"},
			expectedProjectId: "my-project-id",
		},

		// use TOKEN instead of PLUGIN_TOKEN, old drone :-/
		{
			expectedToBeOk:    true,
			Env:               map[string]string{"PLUGIN_ACTION": "deploy", "TOKEN": validGCPKey, "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			expectedProjectId: "my-project-id",
		},

		{
			expectedToBeOk:    false,
			Env:               map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_IMAGE": "my-image"},
			expectedProjectId: "my-project-id",
		},
		{
			expectedToBeOk:    false,
			Env:               map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": validGCPKey, "PLUGIN_SERVICE": "my-service"},
			expectedProjectId: "my-project-id",
		},
		{
			expectedToBeOk:    false,
			Env:               map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_TOKEN": "abcd", "PLUGIN_SERVICE": "my-service"},
			expectedProjectId: "my-project-id",
		},
		{
			expectedToBeOk:    false,
			Env:               map[string]string{"PLUGIN_ACTION": "deploy", "TOKEN": "", "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			expectedProjectId: "my-project-id",
		},
		{
			expectedToBeOk:    false,
			Env:               map[string]string{"PLUGIN_ACTION": "deploy", "TOKEN": "abcde", "PLUGIN_SERVICE": "my-service", "PLUGIN_IMAGE": "my-image"},
			expectedProjectId: "my-project-id",
		},
		{
			expectedToBeOk:    false,
			Env:               map[string]string{"PLUGIN_ACTION": "deploy", "PLUGIN_SERVICE": "my-service"},
			expectedProjectId: "my-project-id",
		},
		{
			expectedToBeOk:    false,
			Env:               map[string]string{"PLUGIN_ACTION": "", "PLUGIN_SERVICE": "my-service"},
			expectedProjectId: "my-project-id",
		},
	} {
		os.Clearenv()

		for k, v := range tst.Env {
			os.Setenv(k, v)
		}

		cfg, err := parseConfig()
		if err != nil && tst.expectedToBeOk == true {
			t.Errorf("parseConfig(  %#v  ) failed, err: %s", tst, err)
			return
		}
		if err == nil && tst.expectedToBeOk == false {
			t.Errorf("parseConfig(  %#v  ) should have failed", tst)
			return
		}
		if !tst.expectedToBeOk {
			continue
		}

		if cfg.Project != tst.expectedProjectId {
			t.Errorf("expected projectID: %s   got: %s", tst.expectedProjectId, cfg.Project)
		}

		if cfg.Token == "" {
			t.Errorf("expected a token, got nothing, tst: %#v", tst)
		}

		for _, e := range tst.expectedEnvSecrets {
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
	}
}
