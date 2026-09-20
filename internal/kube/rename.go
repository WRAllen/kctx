package kube

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RenameContext renames one context in a kubeconfig and updates current-context
// when it points to the renamed context.
func RenameContext(path, oldName, newName string) error {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return fmt.Errorf("context names cannot be blank")
	}
	if oldName == newName {
		return fmt.Errorf("old and new context names are identical")
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve kubeconfig %q: %w", path, err)
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return fmt.Errorf("stat kubeconfig %q: %w", path, err)
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return fmt.Errorf("read kubeconfig %q: %w", path, err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse kubeconfig %q: %w", path, err)
	}
	root := documentRoot(&document)
	if root == nil || root.Kind != yaml.MappingNode {
		return fmt.Errorf("kubeconfig %q does not contain a YAML mapping", path)
	}
	contexts := mappingValue(root, "contexts")
	if contexts == nil || contexts.Kind != yaml.SequenceNode {
		return fmt.Errorf("kubeconfig %q does not contain a contexts list", path)
	}

	var matches []*yaml.Node
	newNameExists := false
	for _, item := range contexts.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		name := mappingValue(item, "name")
		if name == nil || name.Kind != yaml.ScalarNode {
			continue
		}
		if name.Value == oldName {
			matches = append(matches, name)
		}
		if name.Value == newName {
			newNameExists = true
		}
	}
	if len(matches) == 0 {
		return fmt.Errorf("context %q was not found in %q", oldName, path)
	}
	if len(matches) > 1 {
		return fmt.Errorf("context %q appears more than once in %q", oldName, path)
	}
	if newNameExists {
		return fmt.Errorf("context %q already exists in %q", newName, path)
	}

	matches[0].Value = newName
	if current := mappingValue(root, "current-context"); current != nil && current.Kind == yaml.ScalarNode && current.Value == oldName {
		current.Value = newName
	}

	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return fmt.Errorf("encode kubeconfig %q: %w", path, err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("encode kubeconfig %q: %w", path, err)
	}
	if err := replaceFile(resolvedPath, encoded.Bytes(), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write kubeconfig %q: %w", path, err)
	}
	return nil
}

func documentRoot(document *yaml.Node) *yaml.Node {
	if document.Kind == yaml.DocumentNode && len(document.Content) == 1 {
		return document.Content[0]
	}
	return document
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}

func replaceFile(path string, data []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".kctx-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
