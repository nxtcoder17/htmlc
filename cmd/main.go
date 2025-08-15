package main

import (
	_ "embed"
	"errors"
	flag "github.com/spf13/pflag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nxtcoder17/htmlc/cmd/templates"
	"github.com/nxtcoder17/htmlc/examples"
	template_parser "github.com/nxtcoder17/htmlc/pkg/parser/template"

	"github.com/nxtcoder17/fastlog"
)

func isAbs(p string) bool {
	abs, _ := filepath.Abs(p)
	return abs == p
}

func sanitizeConfig(cfg *Config) {
	if !isAbs(cfg.Pages.Input) {
		cfg.Pages.Input = filepath.Join(cfg.WorkingDir, cfg.Pages.Input)
	}

	if !isAbs(cfg.Pages.Output.Dir) {
		cfg.Pages.Output.Dir = filepath.Join(cfg.WorkingDir, cfg.Pages.Output.Dir)
	}

	if !isAbs(cfg.generatorDir) {
		cfg.generatorDir = filepath.Join(cfg.WorkingDir, cfg.generatorDir)
	}

	for i := range cfg.Components {
		if !isAbs(cfg.Components[i].Dir) {
			cfg.Components[i].Dir = filepath.Join(cfg.WorkingDir, cfg.Components[i].Dir)
		}
	}
}

//go:embed templates/pages-generator.gotmpl
var generatorGoCode string

func generator(cfg *Config) error {
	sanitizeConfig(cfg)

	slog.Debug("HERE")

	p, err := template_parser.NewParser(template_parser.Html)
	if err != nil {
		return err
	}

	slog.Debug("parsing components directory")

	// First Sweep
	for _, tc := range cfg.Components {
		if err := p.ParseDir(tc.Dir, cfg.generatorDir, "main", template_parser.ParseOptions{
			GlobPatterns:            tc.Patterns,
			GeneratingForComponents: true,
		}); err != nil {
			return err
		}
	}

	slog.Info("generating pages")
	if err := generatePagesFile(cfg); err != nil {
		return err
	}

	return executor(cfg)
}

func executeCmd(cfg *Config, command string, args ...string) error {
	c := exec.Command(command, args...)
	c.Dir = cfg.generatorDir
	b, err := c.CombinedOutput()
	if err != nil {
		if exerr, ok := err.(*exec.ExitError); ok && exerr.ExitCode() != 0 {
			fmt.Printf("%s\n", b)
		}
		return err
	}

	if debug {
		fmt.Printf("%s\n", b)
	}

	return nil
}

func executor(cfg *Config) error {
	if _, err := os.Stat(filepath.Join(cfg.generatorDir, "go.mod")); err != nil && errors.Is(err, os.ErrNotExist) {
		if err := executeCmd(cfg, "go", "mod", "init", "github.com/nxtcoder17/htmlc.cli"); err != nil {
			return err
		}
	}

	if err := executeCmd(cfg, "go", "get", "github.com/nxtcoder17/htmlc@"+Version); err != nil {
		return err
	}

	if err := executeCmd(cfg, "go", "mod", "tidy"); err != nil {
		return err
	}

	if err := executeCmd(cfg, "go", "run", "./"); err != nil {
		return err
	}

	if !debug {
		slog.Debug("HERE, deleting generator directory", "debug", debug)
		if err := os.RemoveAll(cfg.generatorDir); err != nil {
			return err
		}
	}

	return nil
}

func generatePagesFile(cfg *Config) error {
	rel, err := filepath.Rel(cfg.generatorDir, cfg.Pages.Input)
	if err != nil {
		return err
	}

	b2, err := templates.ParseBytes([]byte(generatorGoCode), map[string]any{
		"package":         "main",
		"input_pages_dir": rel,

		"output_pages_dir":     cfg.Pages.Output.Dir,
		"output_pages_package": cfg.Pages.Output.Package,

		"gen_go_code": cfg.Pages.Output.Go,
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(cfg.generatorDir, "generator.go"), b2, 0o644); err != nil {
		return err
	}

	return nil
}

var (
	debug   bool
	Version string
)

func showHelp() {
	fmt.Println("must specify, one command at least [init|generate]")
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func subCommandInit() error {
	if pathExists("htmlc.yml") || pathExists("components") || pathExists("pages") {
		return fmt.Errorf("htmlc is already initialized as htmlc.yml | components | pages directory already exists")
	}

	return fs.WalkDir(examples.ExamplesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			return nil
		}
		b, err := fs.ReadFile(examples.ExamplesFS, path)
		if err != nil {
			return err
		}

		return os.WriteFile(path, b, 0o644)
	})
}

func main() {
	f := flag.String("config", "htmlc.yml", "--config")
	flag.BoolVar(&debug, "debug", false, "--debug")
	flag.Parse()

	fmt.Println("DEBUG", debug)
	logger := fastlog.New(fastlog.WithoutTimestamp(), fastlog.ShowDebugLogs(debug))
	slog.SetDefault(logger.Slog())

	if len(flag.CommandLine.Args()) == 0 {
		showHelp()
		os.Exit(1)
	}

	cmd := flag.CommandLine.Arg(0)

	switch cmd {
	case "init":
		{
			if err := subCommandInit(); err != nil {
				slog.Error("failed to execute `init`", "err", err)
				os.Exit(1)
			}
		}
	case "generate":
		{
			c, err := ConfigFromFile(*f)
			if err != nil {
				panic(err)
			}

			if err := generator(c); err != nil {
				logger.Error("failed to generate pages, got", "err", err)
				os.Exit(1)
			}
		}
	}
}
