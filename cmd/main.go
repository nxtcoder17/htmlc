package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

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

	for i := range cfg.Templates {
		if !isAbs(cfg.Templates[i].Dir) {
			cfg.Templates[i].Dir = filepath.Join(cfg.WorkingDir, cfg.Templates[i].Dir)
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
	for _, tc := range cfg.Templates {
		if err := p.ParseDir(tc.Dir, cfg.generatorDir, "main", template_parser.ParseOptions{
			GlobPatterns: tc.Patterns,
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

func main() {
	f := flag.String("config", "htmlc.yml", "--config")
	flag.BoolVar(&debug, "debug", false, "--debug")
	flag.Parse()

	c, err := ConfigFromFile(*f)
	if err != nil {
		panic(err)
	}

	if err := generator(c); err != nil {
		panic(err)
	}
}
