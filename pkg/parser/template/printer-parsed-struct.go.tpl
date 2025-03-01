package {{ .Package }}
{{- "\n" -}}
{{- $imports := .Imports }}
{{- $structs := .Structs }}

{{- if $imports }}
{{- "import ("}}
"fmt"
"strings"
{{- range $imports }}
{{ . | quote | indent 2 }}
{{- end }}
{{ ")"}}
{{- "\n" }}
{{- end}}

func init() {
  {{- if .GeneratingForComponents }}
  {{- range $structs }}
  Components[{{.Name | lowercase | quote}}] = func(attr map[string]any) (Component, error) {
    return New{{.Name}}(attr)
  }

  {{- end }}
  {{- end }}
  
  {{.ParseFuncName}}()
}

func {{.ParseFuncName}}() error {
  _, err := Template.Parse(`{{.InputTemplate}}`)
  return err
}

{{- range $structs }}
type {{.Name}} struct {
  {{- range $v := .Fields }}
  {{ $v.Name }} {{ $v.Type }} {{$v.Tag}}
  {{- end }}

  // raw field contains all the
  // - known attributes (i.e. those defined above this line)
  // - and unkwnown ones (props) that are passed in html
  raw map[string]any `json:"-"`
}

func New{{.Name}}(attrs map[string]any) (*{{.Name}}, error) {
	b, err := json.Marshal(attrs)
	if err != nil {
		return nil, err
	}

	var s {{.Name}}

	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}

	known := map[string]any{
    {{- range $v := .Fields }}
    {{ $v.JsonName | quote }}: {{ printf "attrs[%s]," ($v.JsonName | quote) }}
    {{- end }}
  }

  var unknown []string

	for k, v := range attrs {
		if _, ok := known[k]; !ok {
		  unknown = append(unknown, fmt.Sprintf("%s=%q", k, v))
		}
	}

	s.raw = make(map[string]any, len(known) + 1)
  for k, v := range known {
    s.raw[k] = v
  }

  _ = strings.ToLower


  s.raw["props"] = template.HTMLAttr(strings.Join(unknown, " "))

	return &s, nil
}

func (n *{{.Name}}) TemplateName() string {
  return {{.FromTemplate | quote}}
}

func (n *{{.Name}}) Validate() error {
  validate := validator.New(validator.WithRequiredStructEnabled())
  return validate.Struct(n)
}

func (n *{{.Name}}) Render(w io.Writer) error {
  if err := n.Validate(); err != nil {
    return err
  }

  if n.raw == nil {
    b, err := json.Marshal(n)
	  if err != nil {
		  return err
	  }

    n.raw = make(map[string]any)
	  if err := json.Unmarshal(b, &n.raw); err != nil {
		  return err
	  }
  }


  {{- /* return Template.ExecuteTemplate(w, n.TemplateName(), m) */}}
  return Template.ExecuteTemplate(w, n.TemplateName(), n.raw)
}

{{- end }}
