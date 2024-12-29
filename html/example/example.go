package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/nxtcoder17/go-template/pkg/parser/html"
)

type example struct {
	Class string `json:"class"`
}

// Render implements Component.
func (e *example) Render(t *template.Template, w io.Writer) error {
	t, err := t.Parse(`
{{ define "example" }}
<div class="example {{.class}}">
  <span>Example</span>
</div>
{{end}}
`)
	if err != nil {
		return err
	}

	b, err := json.Marshal(*e)
	if err != nil {
		return err
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	return t.ExecuteTemplate(w, e.TemplateName(), m)
}

// TemplateName implements Component.
func (e *example) TemplateName() string {
	return "example"
}

var _ html.Component = (*example)(nil)

func main() {
	reader, err := os.Open("./modules/auth/internal/transport/web/templates/pages/register.html")
	if err != nil {
		panic(err)
	}
	// tokenizeHTML(reader)

	t := template.New("sample")

	components := map[string]html.Component{
		"componentinput": &example{},
	}

	if err := html.Parse(html.Params{
		Input:    reader,
		Output:   os.Stdout,
		Template: t,
		GetComponent: func(name string, attrs map[string]any) (html.Component, error) {
			if v, ok := components[name]; ok {
				b, err := json.Marshal(attrs)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(b, &v); err != nil {
					return nil, err
				}

				return v, nil
			}
			return nil, fmt.Errorf("unknown component (%s)", name)
		},
	}); err != nil {
		panic(err)
	}
}
