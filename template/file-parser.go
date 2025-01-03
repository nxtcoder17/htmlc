package template

import (
	"fmt"
	"text/template"
)

type FileParser struct {
	t       *template.Template
	Content string
}

func (fp *FileParser) Parse() (parsedTmpl string, imports []string, structs []Struct, err error) {
	result := make([]Struct, 0, len(fp.t.Templates()))
	for _, v := range fp.t.Templates() {
		if v.Name() == "t:parser" {
			continue
		}

		sname := generateStructName(v.Name())

		s, err := structFromTemplate(sname, v)
		if err != nil {
			return "", nil, nil, err
		}
		s.FromTemplate = v.Name()

		result = append(result, s)
		imports = append(imports, s.Imports...)
	}

	imports = append(imports,
		"github.com/go-playground/validator/v10",
		"io",
		"encoding/json",
	)

	return fp.Content, imports, result, nil
}

func NewFileParser(content string, defaultStructName string, preProcess ...func(string) (string, error)) (*FileParser, error) {
	t := template.New("t:parser")
	funcs := template.FuncMap{
		paramLabel: func(key, value string) string {
			return "/* comment */"
		},
	}

	t.Funcs(funcs)

	t, err := t.Parse(fixParamComments(content))
	if err != nil {
		return nil, err
	}

	if len(t.Templates()) == 1 {
		content = fmt.Sprintf(`{{- define "%s"}}
%s

{{- end }}`, defaultStructName, content)
	}

	t = template.New("t:parser")
	t.Funcs(funcs)
	t.Parse(content)

	return &FileParser{
		t:       t,
		Content: content,
	}, nil
}
