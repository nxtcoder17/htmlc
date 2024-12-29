package {{ .Package }}
{{- "\n" -}}
{{- $imports := .Imports }}
{{- $structs := .Structs }}

{{- if $imports }}
{{- "import ("}}
{{- range $imports }}
{{ . | quote | indent 2 }}
{{- end }}
{{ ")"}}
{{- "\n" }}
{{- end}}

func init() {
  {{- range $structs }}
  register({{.Name | quote }}, &{{.Name}}{}, {{$.ParseFuncName}})
  {{- end }}
}

func {{.ParseFuncName}}(t *template.Template) error {
  _, err := t.Parse(`{{.InputTemplate}}`)
  return err
}

{{- range $structs }}
type {{.Name}} struct {
  {{- range $v := .Fields }}
  {{ $v.Name }} {{ $v.Type }} {{$v.Tag}}
  {{- end }}
}

func (n *{{.Name}}) TemplateName() string {
  return {{.FromTemplate | quote}}
}

func (n *{{.Name}}) Validate() error {
  validate := validator.New(validator.WithRequiredStructEnabled())
  return validate.Struct(n)
}

func (n *{{.Name}}) Render(t *template.Template, w io.Writer) error {
  if err := n.Validate(); err != nil {
    return err
  }

  b, err := json.Marshal(*n)
	if err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

  return t.ExecuteTemplate(w, n.TemplateName(), m)
}

{{- end }}
