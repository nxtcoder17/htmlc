package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/nxtcoder17/htmlc/examples"
	template_parser "github.com/nxtcoder17/htmlc/pkg/parser/template"
)

func isAbs(p string) bool {
	abs, _ := filepath.Abs(p)
	return abs == p
}

func findGoModFile() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	return filepath.Clean(string(out)), nil
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

//go:embed pages-generator.gotmpl
var pagesGenerator string

//go:embed pages-route.go.tpl
var pagesRouter string

func generator(cfg *Config) error {
	sanitizeConfig(cfg)

	p, err := template_parser.NewParser(template_parser.Html)
	if err != nil {
		return err
	}

	// First Sweep
	for _, tc := range cfg.Components {
		if err := p.ParseDir(tc.Dir, cfg.generatorDir, "main", template_parser.ParseOptions{
			GlobPatterns:            tc.Patterns,
			GeneratingForComponents: true,
		}); err != nil {
			return err
		}
	}

	if err := generatePagesFile(cfg); err != nil {
		return err
	}

	return executor(cfg)
}

func executor(cfg *Config) error {
	b, err := exec.Command("go", "run", cfg.generatorDir).CombinedOutput()
	if err != nil {
		if exerr, ok := err.(*exec.ExitError); ok && exerr.ExitCode() != 0 {
			fmt.Printf("%s\n", b)
		}
		return err
	}

	if debug {
		fmt.Printf("%s\n", b)
	}

	if !debug {
		if err := os.RemoveAll(cfg.generatorDir); err != nil {
			return err
		}
	}

	return nil
}

func generatePagesFile(cfg *Config) error {
	t := template.New("pages-generator").Funcs(template.FuncMap{
		"indent": func(indent int, str string) string {
			return strings.Repeat(" ", indent) + str
		},
		"trim": func(str string) string {
			return strings.TrimSpace(str)
		},
		"quote": func(str string) string {
			return strconv.Quote(str)
		},

		"squote": func(str string) string {
			str = strconv.Quote(str)
			return "'" + str[1:len(str)-1] + "'"
		},
	})

	t, err := t.Parse(pagesGenerator)
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(cfg.generatorDir, cfg.Pages.Input)
	if err != nil {
		panic(err)
	}

	out, err := os.Create(filepath.Join(cfg.generatorDir, "pages_generator.go"))
	if err != nil {
		return err
	}

	return t.ExecuteTemplate(out, t.Name(), map[string]any{
		"package":         "main",
		"input_pages_dir": rel,

		"output_pages_dir":     cfg.Pages.Output.Dir,
		"output_pages_package": cfg.Pages.Output.Package,
	})
}

var debug bool

func showHelp() {
	fmt.Println("must specify, one command at least [init|generate]")
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func subCommandInit() error {
	if pathExists("htmlc.yml") || pathExists("components") || pathExists("pages") {
		return fmt.Errorf("htmlc is already initialized as htmlc.yml file, components, or pages directory already exists")
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
				panic(err)
			}
		}
	}
}
