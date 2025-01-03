package {{ .package }}

import (
  {{.template_import | quote}}
  "strings"
  "io"
  "fmt"
  "encoding/json"
)

var Template *template.Template = template.New("template:{{.package}}")

type GetComponentFn func(attr map[string]any) (Component, error)
var Components map[string]GetComponentFn = make(map[string]GetComponentFn)

type Component interface {
  Render(t *template.Template, w io.Writer) error
}

func register(name string, getc func() Component, parse func(*template.Template) error) {
  if err := parse(Template); err != nil {
    panic(fmt.Errorf("failed to parse template (%s), got error:\n%v", name, err))
  }

  Components[strings.ToLower(name)] = func(attrs map[string]any) (Component, error) {
		c := getc()
		b, err := json.Marshal(attrs)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(b, &c); err != nil {
			return nil, err
		}

		return c, nil
  }
}
