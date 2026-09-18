package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WRAllen/kctx/internal/config"
)

func TestRunFiltersAndMapsEnvironment(t *testing.T) {
	dir := t.TempDir()
	kubeDir := filepath.Join(dir, "kube")
	if err := os.Mkdir(kubeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	kubeconfig := "apiVersion: v1\ncontexts:\n  - name: sh02-prod-minimax\n  - name: dev04\n"
	if err := os.WriteFile(filepath.Join(kubeDir, "config.yaml"), []byte(kubeconfig), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	configData := "kubeconfig_dir: " + kubeDir + "\nalias:\n  name: AMS_ENV\n  values:\n    sh02-prod-minimax: sco-prod-sh02-bmsk8s-02\n"
	if err := os.WriteFile(configPath, []byte(configData), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"--config", configPath, "minimax"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "sh02-prod-minimax") || !strings.Contains(stdout.String(), "sco-prod-sh02-bmsk8s-02") {
		t.Fatalf("unexpected output:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "dev04") {
		t.Fatalf("filter did not exclude dev04:\n%s", stdout.String())
	}
}

func TestConfigHelpReturnsSuccess(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{"config", "--help"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Usage: kctx config") {
		t.Fatalf("help output is missing usage: %s", stderr.String())
	}
}

func TestAliasSetAndClear(t *testing.T) {
	tests := []struct {
		name      string
		clearArgs []string
	}{
		{name: "omitted value", clearArgs: nil},
		{name: "empty value", clearArgs: []string{""}},
		{name: "blank value", clearArgs: []string{"   "}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			var stdout, stderr bytes.Buffer
			setArgs := []string{"alias", "set", "--config", configPath, "dev-cluster", "development"}
			if exitCode := Run(setArgs, strings.NewReader(""), &stdout, &stderr); exitCode != 0 {
				t.Fatalf("set exit code = %d, stderr = %s", exitCode, stderr.String())
			}

			settings, err := config.Load(configPath, "")
			if err != nil {
				t.Fatal(err)
			}
			if settings.Alias.Values["dev-cluster"] != "development" {
				t.Fatalf("alias was not set: %#v", settings.Alias.Values)
			}

			stdout.Reset()
			stderr.Reset()
			clearArgs := []string{"alias", "set", "--config", configPath, "dev-cluster"}
			clearArgs = append(clearArgs, test.clearArgs...)
			if exitCode := Run(clearArgs, strings.NewReader(""), &stdout, &stderr); exitCode != 0 {
				t.Fatalf("clear exit code = %d, stderr = %s", exitCode, stderr.String())
			}

			settings, err = config.Load(configPath, "")
			if err != nil {
				t.Fatal(err)
			}
			if _, exists := settings.Alias.Values["dev-cluster"]; exists {
				t.Fatalf("alias was not cleared: %#v", settings.Alias.Values)
			}
		})
	}
}
