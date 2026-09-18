package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/WRAllen/kctx/internal/config"
	"github.com/WRAllen/kctx/internal/kube"
	"github.com/WRAllen/kctx/internal/output"
	"github.com/WRAllen/kctx/internal/setup"
)

const Version = "0.3.0"

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(stderr, "kctx: determine home directory: %v\n", err)
		return 1
	}

	defaultKubeDir := filepath.Join(home, ".kube")
	defaultConfigPath := filepath.Join(home, ".config", "kctx", "config.yaml")
	legacyPath := filepath.Join(defaultKubeDir, "ams-env-map")
	if len(args) > 0 && args[0] == "config" {
		return runConfig(args[1:], stdin, stdout, stderr, defaultConfigPath, defaultKubeDir, legacyPath)
	}
	if len(args) > 0 && args[0] == "alias" {
		return runAlias(args[1:], stdout, stderr, defaultConfigPath, legacyPath)
	}
	return runList(args, stdout, stderr, defaultConfigPath, defaultKubeDir, legacyPath)
}

func runList(args []string, stdout, stderr io.Writer, defaultConfigPath, defaultKubeDir, legacyPath string) int {
	flags := flag.NewFlagSet("kctx", flag.ContinueOnError)
	flags.SetOutput(stderr)
	kubeDirFlag := flags.String("kube-dir", "", "override the configured kubeconfig directory")
	configPath := flags.String("config", defaultConfigPath, "configuration file")
	jsonOutput := flags.Bool("json", false, "print JSON instead of a table")
	noColor := flags.Bool("no-color", false, "disable colored table headers")
	showVersion := flags.Bool("version", false, "print the version")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: kctx [options] [context keyword]")
		fmt.Fprintln(stderr, "       kctx config [--config path]")
		fmt.Fprintln(stderr, "       kctx alias set [--config path] <context> [value]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "List kubeconfig files and their contexts. The keyword match is case-insensitive.")
		fmt.Fprintln(stderr)
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, Version)
		return 0
	}
	if flags.NArg() > 1 {
		flags.Usage()
		return 2
	}

	settings, err := config.Load(*configPath, legacyPath)
	if err != nil {
		fmt.Fprintf(stderr, "kctx: %v\n", err)
		return 1
	}
	kubeDir := settings.KubeconfigDir
	if kubeDir == "" {
		kubeDir = defaultKubeDir
	}
	if *kubeDirFlag != "" {
		kubeDir = *kubeDirFlag
	}

	query := ""
	if flags.NArg() == 1 {
		query = strings.ToLower(flags.Arg(0))
	}
	contexts, err := kube.Scan(kubeDir)
	if err != nil {
		fmt.Fprintf(stderr, "kctx: %v\n", err)
		return 1
	}

	filtered := make([]kube.Context, 0, len(contexts))
	for _, context := range contexts {
		if query != "" && !strings.Contains(strings.ToLower(context.Name), query) {
			continue
		}
		context.Alias = settings.Alias.Values[context.Name]
		if context.Alias == "" {
			context.Alias = "-"
		}
		filtered = append(filtered, context)
	}

	if len(filtered) == 0 {
		if query == "" {
			fmt.Fprintln(stderr, "kctx: no contexts found")
		} else {
			fmt.Fprintf(stderr, "kctx: no context contains %q\n", flags.Arg(0))
		}
		return 1
	}

	aliasName := settings.Alias.Name
	if aliasName == "" {
		aliasName = config.DefaultAliasName
	}
	if *jsonOutput {
		err = output.JSON(stdout, filtered, aliasName)
	} else {
		err = output.Table(stdout, filtered, aliasName, !*noColor && os.Getenv("NO_COLOR") == "" && isTerminal(stdout))
	}
	if err != nil {
		fmt.Fprintf(stderr, "kctx: write output: %v\n", err)
		return 1
	}
	return 0
}

func runAlias(args []string, stdout, stderr io.Writer, defaultConfigPath, legacyPath string) int {
	if len(args) == 0 || args[0] != "set" {
		fmt.Fprintln(stderr, "Usage: kctx alias set [--config path] <context> [value]")
		return 2
	}

	flags := flag.NewFlagSet("kctx alias set", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", defaultConfigPath, "configuration file to edit")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: kctx alias set [--config path] <context> [value]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Set one context alias. An omitted or blank value clears the alias.")
		fmt.Fprintln(stderr)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() < 1 {
		flags.Usage()
		return 2
	}

	contextName := strings.TrimSpace(flags.Arg(0))
	if contextName == "" {
		fmt.Fprintln(stderr, "kctx alias set: context cannot be blank")
		return 2
	}
	value := strings.TrimSpace(strings.Join(flags.Args()[1:], " "))

	settings, err := config.Load(*configPath, legacyPath)
	if err != nil {
		fmt.Fprintf(stderr, "kctx alias set: %v\n", err)
		return 1
	}
	if settings.Alias.Values == nil {
		settings.Alias.Values = make(map[string]string)
	}
	if value == "" {
		delete(settings.Alias.Values, contextName)
	} else {
		settings.Alias.Values[contextName] = value
	}
	if err := config.Save(*configPath, settings); err != nil {
		fmt.Fprintf(stderr, "kctx alias set: %v\n", err)
		return 1
	}

	if value == "" {
		fmt.Fprintf(stdout, "Cleared alias for %s\n", contextName)
	} else {
		fmt.Fprintf(stdout, "Set alias for %s to %s\n", contextName, value)
	}
	return 0
}

func runConfig(args []string, stdin io.Reader, stdout, stderr io.Writer, defaultConfigPath, defaultKubeDir, legacyPath string) int {
	flags := flag.NewFlagSet("kctx config", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", defaultConfigPath, "configuration file to edit")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: kctx config [--config path]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Interactively configure the kubeconfig directory and alias values.")
		fmt.Fprintln(stderr)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return 2
	}
	if err := setup.Run(stdin, stdout, *configPath, defaultKubeDir, legacyPath); err != nil {
		fmt.Fprintf(stderr, "kctx config: %v\n", err)
		return 1
	}
	return 0
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
