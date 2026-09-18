package kube

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "one.yaml"), `
apiVersion: v1
contexts:
  - name: z-context
  - name: a-context
`)
	writeFile(t, filepath.Join(dir, "invalid.txt"), "not: [valid")
	writeFile(t, filepath.Join(dir, ".hidden.yaml"), "contexts:\n  - name: hidden\n")
	if err := os.Mkdir(filepath.Join(dir, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("Scan() returned %d contexts, want 2", len(got))
	}
	if got[0].Name != "a-context" || got[1].Name != "z-context" {
		t.Fatalf("Scan() returned unexpected order: %#v", got)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
