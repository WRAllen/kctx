package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/WRAllen/kctx/internal/config"
	"github.com/WRAllen/kctx/internal/kube"
)

// Run starts the portable, line-oriented configuration wizard.
func Run(input io.Reader, output io.Writer, configPath, defaultKubeDir, legacyPath string) error {
	current, err := config.Load(configPath, legacyPath)
	if err != nil {
		return err
	}
	if current.KubeconfigDir == "" {
		current.KubeconfigDir = defaultKubeDir
	}

	reader := bufio.NewReader(input)
	fmt.Fprintln(output, "kctx configuration")
	fmt.Fprintln(output, "Press Enter to keep the current value.")
	fmt.Fprintln(output)

	var contexts []kube.Context
	for {
		answer, err := prompt(reader, output, "Kubeconfig directory", current.KubeconfigDir)
		if err != nil {
			return err
		}
		directory, err := absolutePath(answer)
		if err != nil {
			fmt.Fprintf(output, "Invalid path: %v\n", err)
			continue
		}
		contexts, err = kube.Scan(directory)
		if err != nil {
			fmt.Fprintf(output, "Cannot scan that directory: %v\n", err)
			continue
		}
		current.KubeconfigDir = directory
		break
	}

	aliasName, err := prompt(reader, output, "Alias column name", current.Alias.Name)
	if err != nil {
		return err
	}
	aliasName = strings.TrimSpace(aliasName)
	if aliasName == "" {
		aliasName = config.DefaultAliasName
	}
	current.Alias.Name = aliasName
	if current.Alias.Values == nil {
		current.Alias.Values = make(map[string]string)
	}

	names := uniqueContextNames(contexts)
	fmt.Fprintf(output, "Found %d contexts.\n", len(names))
	edit, err := confirm(reader, output, "Configure alias values", true)
	if err != nil {
		return err
	}
	if edit {
		fmt.Fprintln(output, "Press Enter to keep a value; enter - to clear it.")
		for _, name := range names {
			value, err := prompt(reader, output, fmt.Sprintf("%s for %s", current.Alias.Name, name), current.Alias.Values[name])
			if err != nil {
				return err
			}
			value = strings.TrimSpace(value)
			if value == "-" {
				delete(current.Alias.Values, name)
			} else if value != "" {
				current.Alias.Values[name] = value
			}
		}
	}

	if err := config.Save(configPath, current); err != nil {
		return err
	}
	fmt.Fprintf(output, "\nSaved configuration to %s\n", configPath)
	return nil
}

func prompt(reader *bufio.Reader, output io.Writer, label, current string) (string, error) {
	if current == "" {
		fmt.Fprintf(output, "%s: ", label)
	} else {
		fmt.Fprintf(output, "%s [%s]: ", label, current)
	}
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	if err == io.EOF && line == "" {
		return "", io.EOF
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return current, nil
	}
	return line, nil
}

func confirm(reader *bufio.Reader, output io.Writer, label string, defaultYes bool) (bool, error) {
	suffix := "[Y/n]"
	if !defaultYes {
		suffix = "[y/N]"
	}
	fmt.Fprintf(output, "%s %s: ", label, suffix)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	if err == io.EOF && line == "" {
		return false, io.EOF
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "" {
		return defaultYes, nil
	}
	return answer == "y" || answer == "yes", nil
}

func absolutePath(path string) (string, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "~" || strings.HasPrefix(cleaned, "~/") || strings.HasPrefix(cleaned, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if cleaned == "~" {
			cleaned = home
		} else {
			cleaned = filepath.Join(home, cleaned[2:])
		}
	}
	cleaned = filepath.Clean(cleaned)
	return filepath.Abs(cleaned)
}

func uniqueContextNames(contexts []kube.Context) []string {
	seen := make(map[string]struct{})
	for _, context := range contexts {
		seen[context.Name] = struct{}{}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
