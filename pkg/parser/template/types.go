package template

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

type Struct struct {
	Name         string
	Imports      []string
	Fields       []StructField
	FromTemplate string
}

func (st *Struct) String() (string, error) {
	t := template.New("parse").Funcs(template.FuncMap{
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
		"titlecase": func(str string) string {
			return strings.ToTitle(str)
		},
	})
	t, err := t.Parse(`
{{- if .Imports }}
{{- "import ("}}
{{- range .Imports }}
{{ . | quote | indent 2 }}
{{- end }}
{{ ")"}}
{{- "\n" }}
{{- end}}
type {{.Name}} struct {
  {{- range $v := .Fields }}
  {{ $v.Name | titlecase }} {{ $v.Type }}
  {{- end }}
}
`)
	if err != nil {
		return "", err
	}

	b := new(bytes.Buffer)
	if err := t.Execute(b, st); err != nil {
		return "", err
	}

	return strings.TrimSpace(b.String()), nil
}

type StructField struct {
	Name     string
	Type     string
	Package  *string
	JsonName string
	Tag      string
}

func toFieldName(str string) string {
	l := len(str)
	if strings.HasSuffix(str, "?") {
		l -= 1
	}

	res := new(strings.Builder)
	for i := 0; i < l; i++ {
		if i == 0 {
			res.WriteString(strings.ToUpper(string(str[i])))
			continue
		}
		if str[i] == '/' || str[i] == '-' || str[i] == '_' {
			i += 1
			res.WriteString(strings.ToUpper(string(str[i])))
			continue
		}
		res.WriteByte(str[i])
	}

	return res.String()
}

func toStructField(varName, varType string) StructField {
	fieldName := toFieldName(varName)

	var tag, jsonName string
	switch {
	case strings.HasSuffix(varName, "?"):
		// Optional Variable
		jsonName = varName[:len(varName)-1]
		tag = fmt.Sprint("`", fmt.Sprintf(`json:"%s"`, jsonName), "`")
	default:
		jsonName = varName
		tag = fmt.Sprint("`", fmt.Sprintf(`json:"%s" validate:"required"`, jsonName), "`")
	}

	return StructField{
		Name:     fieldName,
		Type:     varType,
		Tag:      tag,
		JsonName: jsonName,
	}
}

func generateStructName(str string) string {
	res := new(strings.Builder)
	for i := 0; i < len(str); i++ {
		if i == 0 {
			res.WriteString(strings.ToUpper(string(str[i])))
			continue
		}
		if str[i] == '/' || str[i] == '-' || str[i] == '.' {
			i += 1
			res.WriteString(strings.ToUpper(string(str[i])))
			continue
		}
		res.WriteByte(str[i])
	}

	return res.String()
}
