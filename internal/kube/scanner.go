package kube

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Context describes one context found in one kubeconfig file.
type Context struct {
	Name     string `json:"context"`
	Filepath string `json:"filepath"`
	Alias    string `json:"alias"`
}

type kubeconfig struct {
	Contexts []struct {
		Name string `yaml:"name"`
	} `yaml:"contexts"`
}

// Scan reads regular files directly under dir and returns every kube context found.
func Scan(dir string) ([]Context, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read kube directory %q: %w", dir, err)
	}

	var contexts []Context
	seen := make(map[string]struct{})

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}

		names, err := ContextNames(path)
		if err != nil {
			continue
		}

		for _, name := range names {
			key := name + "\x00" + path
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			contexts = append(contexts, Context{Name: name, Filepath: path})
		}
	}

	sort.Slice(contexts, func(i, j int) bool {
		if contexts[i].Name == contexts[j].Name {
			return contexts[i].Filepath < contexts[j].Filepath
		}
		return contexts[i].Name < contexts[j].Name
	})

	return contexts, nil
}

// ContextNames returns the context names declared by one kubeconfig file.
func ContextNames(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read kubeconfig %q: %w", path, err)
	}

	var config kubeconfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse kubeconfig %q: %w", path, err)
	}

	names := make([]string, 0, len(config.Contexts))
	for _, item := range config.Contexts {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}
