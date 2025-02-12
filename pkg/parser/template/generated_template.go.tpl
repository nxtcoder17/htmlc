package {{ .package }}

import (
  {{.template_import | quote}}
  "io"
)

var Template *template.Template = template.New("template:{{.package}}")

type GetComponentFn func(attr map[string]any) (Component, error)
var Components map[string]GetComponentFn = make(map[string]GetComponentFn)

type Component interface {
  Render(w io.Writer) error
}

func must[T any](v T, err error) T {
  if err != nil {
    panic(err)
  }

  return v
}
