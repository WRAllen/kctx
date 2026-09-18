package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WRAllen/kctx/internal/config"
)

func TestRunWithWindowsLineEndings(t *testing.T) {
	dir := t.TempDir()
	kubeDir := filepath.Join(dir, "kube")
	if err := os.Mkdir(kubeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	kubeconfig := "apiVersion: v1\ncontexts:\n  - name: a800\n  - name: dev04\n"
	if err := os.WriteFile(filepath.Join(kubeDir, "config.yaml"), []byte(kubeconfig), 0o600); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(dir, "config", "config.yaml")
	input := strings.NewReader("\r\nAMS_ENV\r\ny\r\nsco-dev-sh01-bmsk8s\r\nsco-dev-04\r\n")
	var output bytes.Buffer
	if err := Run(input, &output, configPath, kubeDir, filepath.Join(dir, "missing")); err != nil {
		t.Fatal(err)
	}

	got, err := config.Load(configPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.KubeconfigDir != kubeDir || got.Alias.Name != "AMS_ENV" {
		t.Fatalf("unexpected saved config: %#v", got)
	}
	if got.Alias.Values["a800"] != "sco-dev-sh01-bmsk8s" || got.Alias.Values["dev04"] != "sco-dev-04" {
		t.Fatalf("unexpected alias values: %#v", got.Alias.Values)
	}
}
