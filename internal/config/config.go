package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultAliasName = "ALIAS"

type Alias struct {
	Name   string            `yaml:"name"`
	Values map[string]string `yaml:"values,omitempty"`
}

type Config struct {
	KubeconfigDir string `yaml:"kubeconfig_dir,omitempty"`
	Alias         Alias  `yaml:"alias"`
}

type diskConfig struct {
	KubeconfigDir string            `yaml:"kubeconfig_dir,omitempty"`
	Alias         Alias             `yaml:"alias"`
	Environments  map[string]string `yaml:"environments,omitempty"`
}

// Load reads the current YAML config. The old environments field and the
// original context=environment map are supported for automatic migration.
func Load(path, legacyPath string) (Config, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		var disk diskConfig
		if err := yaml.Unmarshal(data, &disk); err != nil {
			return Config{}, fmt.Errorf("parse config %q: %w", path, err)
		}
		result := Config{KubeconfigDir: disk.KubeconfigDir, Alias: disk.Alias}
		if len(result.Alias.Values) == 0 && len(disk.Environments) > 0 {
			result.Alias.Name = "AMS_ENV"
			result.Alias.Values = disk.Environments
		}
		normalize(&result)
		return result, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	values, err := loadLegacy(legacyPath)
	if err != nil {
		return Config{}, err
	}
	result := Config{Alias: Alias{Name: DefaultAliasName, Values: values}}
	if len(values) > 0 {
		result.Alias.Name = "AMS_ENV"
	}
	return result, nil
}

func Save(path string, value Config) error {
	normalize(&value)
	data, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

func normalize(value *Config) {
	value.KubeconfigDir = strings.TrimSpace(value.KubeconfigDir)
	value.Alias.Name = strings.TrimSpace(value.Alias.Name)
	if value.Alias.Name == "" {
		value.Alias.Name = DefaultAliasName
	}
	if value.Alias.Values == nil {
		value.Alias.Values = make(map[string]string)
	}
}

func loadLegacy(path string) (map[string]string, error) {
	result := make(map[string]string)
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read legacy map %q: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		contextName, alias, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		contextName = strings.TrimSpace(contextName)
		alias = strings.TrimSpace(alias)
		if contextName != "" && alias != "" {
			result[contextName] = alias
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read legacy map %q: %w", path, err)
	}
	return result, nil
}
