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

func TestContextRenameUpdatesFileAndAlias(t *testing.T) {
	dir := t.TempDir()
	kubeDir := filepath.Join(dir, "kube")
	if err := os.Mkdir(kubeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	kubeconfigPath := filepath.Join(kubeDir, "new-cluster.yaml")
	kubeconfig := "apiVersion: v1\ncontexts:\n  - name: old-context\n    context:\n      cluster: cluster-one\ncurrent-context: old-context\n"
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfig), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	configData := "kubeconfig_dir: " + kubeDir + "\nalias:\n  name: ENVIRONMENT\n  values:\n    old-context: development\n"
	if err := os.WriteFile(configPath, []byte(configData), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	args := []string{"context", "rename", "--config", configPath, "new-cluster.yaml", "new-context"}
	if exitCode := Run(args, strings.NewReader(""), &stdout, &stderr); exitCode != 0 {
		t.Fatalf("rename exit code = %d, stderr = %s", exitCode, stderr.String())
	}

	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "old-context") || !strings.Contains(string(data), "current-context: new-context") {
		t.Fatalf("kubeconfig was not renamed correctly:\n%s", data)
	}
	settings, err := config.Load(configPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := settings.Alias.Values["old-context"]; exists {
		t.Fatalf("old alias key remains: %#v", settings.Alias.Values)
	}
	if settings.Alias.Values["new-context"] != "development" {
		t.Fatalf("alias was not migrated: %#v", settings.Alias.Values)
	}
}

func TestContextRenameRejectsMultipleContextsInFile(t *testing.T) {
	dir := t.TempDir()
	content := "contexts:\n  - name: context-one\n  - name: context-two\n"
	if err := os.WriteFile(filepath.Join(dir, "multiple.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configData := "kubeconfig_dir: " + dir + "\nalias:\n  name: ENVIRONMENT\n"
	if err := os.WriteFile(configPath, []byte(configData), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	args := []string{"context", "rename", "--config", configPath, "multiple.yaml", "new-context"}
	if exitCode := Run(args, strings.NewReader(""), &stdout, &stderr); exitCode == 0 {
		t.Fatalf("rename unexpectedly succeeded: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "contains multiple contexts") {
		t.Fatalf("unexpected multiple-context error: %s", stderr.String())
	}
}

func TestDeleteRemovesFileAndAlias(t *testing.T) {
	dir := t.TempDir()
	kubeDir := filepath.Join(dir, "kube")
	if err := os.Mkdir(kubeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	kubeconfigPath := filepath.Join(kubeDir, "dev-cluster.yaml")
	if err := os.WriteFile(kubeconfigPath, []byte("contexts:\n  - name: dev-cluster\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	configData := "kubeconfig_dir: " + kubeDir + "\nalias:\n  name: ENVIRONMENT\n  values:\n    dev-cluster: development\n"
	if err := os.WriteFile(configPath, []byte(configData), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	args := []string{"delete", "--config", configPath, "dev-cluster"}
	if exitCode := Run(args, strings.NewReader("y\r\n"), &stdout, &stderr); exitCode != 0 {
		t.Fatalf("delete exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if _, err := os.Stat(kubeconfigPath); !os.IsNotExist(err) {
		t.Fatalf("kubeconfig still exists or stat failed unexpectedly: %v", err)
	}
	settings, err := config.Load(configPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := settings.Alias.Values["dev-cluster"]; exists {
		t.Fatalf("alias was not removed: %#v", settings.Alias.Values)
	}
}

func TestDeleteCancellationKeepsFile(t *testing.T) {
	dir := t.TempDir()
	kubeconfigPath := filepath.Join(dir, "dev-cluster.yaml")
	if err := os.WriteFile(kubeconfigPath, []byte("contexts:\n  - name: dev-cluster\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configData := "kubeconfig_dir: " + dir + "\nalias:\n  name: ENVIRONMENT\n"
	if err := os.WriteFile(configPath, []byte(configData), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	args := []string{"delete", "--config", configPath, "dev-cluster"}
	if exitCode := Run(args, strings.NewReader("n\n"), &stdout, &stderr); exitCode != 0 {
		t.Fatalf("cancel exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if _, err := os.Stat(kubeconfigPath); err != nil {
		t.Fatalf("cancelled deletion removed the file: %v", err)
	}
	if !strings.Contains(stdout.String(), "Cancelled") {
		t.Fatalf("cancel output missing: %s", stdout.String())
	}
}

func TestDeleteForceSkipsConfirmation(t *testing.T) {
	dir := t.TempDir()
	kubeconfigPath := filepath.Join(dir, "dev-cluster.yaml")
	if err := os.WriteFile(kubeconfigPath, []byte("contexts:\n  - name: dev-cluster\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("alias:\n  name: ENVIRONMENT\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	args := []string{"delete", "--force", "--config", configPath, "--kube-dir", dir, "dev-cluster"}
	if exitCode := Run(args, strings.NewReader(""), &stdout, &stderr); exitCode != 0 {
		t.Fatalf("forced delete exit code = %d, stderr = %s", exitCode, stderr.String())
	}
	if _, err := os.Stat(kubeconfigPath); !os.IsNotExist(err) {
		t.Fatalf("forced deletion did not remove file: %v", err)
	}
}

func TestDeleteRejectsFileWithMultipleContexts(t *testing.T) {
	dir := t.TempDir()
	kubeconfigPath := filepath.Join(dir, "shared.yaml")
	content := "contexts:\n  - name: dev-cluster\n  - name: prod-cluster\n"
	if err := os.WriteFile(kubeconfigPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("alias:\n  name: ENVIRONMENT\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	args := []string{"delete", "--force", "--config", configPath, "--kube-dir", dir, "dev-cluster"}
	if exitCode := Run(args, strings.NewReader(""), &stdout, &stderr); exitCode == 0 {
		t.Fatalf("delete unexpectedly succeeded: %s", stdout.String())
	}
	if _, err := os.Stat(kubeconfigPath); err != nil {
		t.Fatalf("protected file was removed: %v", err)
	}
	if !strings.Contains(stderr.String(), "contains other or duplicate contexts") {
		t.Fatalf("unexpected refusal message: %s", stderr.String())
	}
}
