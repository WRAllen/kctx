package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := "kubeconfig_dir: /tmp/kube\nalias:\n  name: CLUSTER\n  values:\n    a800: sco-dev-sh01-bmsk8s\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.KubeconfigDir != "/tmp/kube" || got.Alias.Name != "CLUSTER" || got.Alias.Values["a800"] != "sco-dev-sh01-bmsk8s" {
		t.Fatalf("unexpected config: %#v", got)
	}
}

func TestLoadOldYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("environments:\n  a800: sco-dev-sh01-bmsk8s\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Alias.Name != "AMS_ENV" || got.Alias.Values["a800"] != "sco-dev-sh01-bmsk8s" {
		t.Fatalf("old YAML was not migrated: %#v", got)
	}
}

func TestLoadLegacyFallback(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "ams-env-map")
	if err := os.WriteFile(legacy, []byte("# comment\na800=sco-dev-sh01-bmsk8s\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(filepath.Join(dir, "missing.yaml"), legacy)
	if err != nil {
		t.Fatal(err)
	}
	if got.Alias.Name != "AMS_ENV" || got.Alias.Values["a800"] != "sco-dev-sh01-bmsk8s" {
		t.Fatalf("unexpected legacy environment map: %#v", got)
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	want := Config{
		KubeconfigDir: "/tmp/kube",
		Alias: Alias{
			Name:   "AMS_ENV",
			Values: map[string]string{"a800": "sco-dev-sh01-bmsk8s"},
		},
	}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.KubeconfigDir != want.KubeconfigDir || got.Alias.Name != want.Alias.Name || got.Alias.Values["a800"] != want.Alias.Values["a800"] {
		t.Fatalf("round trip mismatch: got %#v, want %#v", got, want)
	}
}
