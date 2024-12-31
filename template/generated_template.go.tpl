package {{ .package }}

import (
  {{.template_import | quote}}
  "strings"
  "io"
  "fmt"
)

var Template *template.Template = template.New("template:{{.package}}")
var Components map[string]Component = make(map[string]Component)

type Component interface {
  Render(t *template.Template, w io.Writer) error
}

func register(name string, c Component, parse func(*template.Template) error) {
  if err := parse(Template); err != nil {
    panic(fmt.Errorf("failed to parse template (%s), got error:\n%v", name, err))
  }

  Components[strings.ToLower(name)] = c
}

