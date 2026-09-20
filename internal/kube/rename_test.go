package kube

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenameContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `apiVersion: v1
kind: Config
clusters:
  - name: cluster-one
    cluster:
      server: https://example.invalid
users:
  - name: user-one
    user:
      token: keep-this-secret
contexts:
  - name: old-context
    context:
      cluster: cluster-one
      user: user-one
  - name: other-context
    context:
      cluster: cluster-one
      user: user-one
current-context: old-context
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RenameContext(path, "old-context", "new-context"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)
	for _, expected := range []string{"name: new-context", "name: other-context", "current-context: new-context", "token: keep-this-secret"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("renamed kubeconfig does not contain %q:\n%s", expected, result)
		}
	}
	if strings.Contains(result, "old-context") {
		t.Fatalf("old context remains in kubeconfig:\n%s", result)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("file mode = %o, want 600", got)
	}
}

func TestRenameContextRejectsCollision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "contexts:\n  - name: old-context\n  - name: new-context\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RenameContext(path, "old-context", "new-context"); err == nil {
		t.Fatal("RenameContext() succeeded with a duplicate target context")
	}
}
