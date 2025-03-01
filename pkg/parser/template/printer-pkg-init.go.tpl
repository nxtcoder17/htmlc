package {{ .Package }}

import (
  {{.TemplateImport | quote}}
  {{- if .GeneratingForComponents }}
  "io"
  {{- end }}
)

var Template *template.Template = template.New("template:{{.Package}}")

{{- if .GeneratingForComponents }}
type GetComponentFn func(attr map[string]any) (Component, error)
var Components map[string]GetComponentFn = make(map[string]GetComponentFn)

type Component interface {
  Render(w io.Writer) error
}
{{- end }}

