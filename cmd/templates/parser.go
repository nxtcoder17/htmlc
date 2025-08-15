package templates

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

func ParseBytes(b []byte, values any) ([]byte, error) {
	t := template.New("parse-bytes")
	t.Funcs(template.FuncMap{
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
	})

	if _, err := t.Parse(string(b)); err != nil {
		return nil, err
	}
	out := new(bytes.Buffer)
	if err := t.ExecuteTemplate(out, t.Name(), values); err != nil {
		return nil, fmt.Errorf("failed to execute template")
	}
	return out.Bytes(), nil
}
