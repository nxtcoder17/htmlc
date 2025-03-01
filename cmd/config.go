package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"sigs.k8s.io/yaml"
)

type Config struct {
	WorkingDir string

	Components []Components `json:"components"`
	Pages      Pages        `json:"pages,omitempty"`

	generatorDir string
}

type Pages struct {
	Input  string `json:"input"`
	Output struct {
		Package string `json:"pkg" validate:"required"`
		Dir     string `json:"dir" validate:"required"`
	} `json:"output" validate:"required"`
}

type Components struct {
	Dir string `json:"dir" validate:"required"`

	// Patterns must follow [guidelines](https://pkg.go.dev/path/filepath#Match)
	// htmlc does recursive matching of patterns
	Patterns []string `json:"patterns"`
}

func ConfigFromFile(file string) (*Config, error) {
	fi, err := os.Stat(file)
	if err != nil || fi.IsDir() {
		return nil, fmt.Errorf("failed to find htmlc configuration file (%s), got err %w", file, err)
	}

	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(&cfg); err != nil {
		return nil, err
	}

	s, err := filepath.Abs(filepath.Dir(file))
	if err != nil {
		return nil, err
	}

	cfg.WorkingDir = s

	cfg.generatorDir = cfg.Pages.Output.Dir + ".tt"

	return &cfg, nil
}
