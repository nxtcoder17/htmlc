package template

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func Test_generateStruct(t *testing.T) {
	type args struct {
		tmpl string
	}
	tests := []struct {
		name    string
		args    args
		want    []Struct
		wantErr bool
	}{
		{
			name: "1. simple variables",
			args: args{
				tmpl: /*gotmpl*/ `
{{- /* @param Name string */}}
{{- /* @param Message string */}}
Hello, {{ .Name }}!
Message: {{.Message}}
`,
			},
			want: []Struct{
				{
					Name: defaultStructName,
					Fields: []StructField{
						{Name: "Name", Type: "string"},
						{Name: "Message", Type: "string"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "2. with if expressions",
			args: args{
				tmpl: /*gotmpl*/ `
{{- define "Sample" -}}
{{- /* @param Name string */}}
{{- /* @param Message string */}}
{{- /* @param ShowMessage bool */}}
Hello, {{ .Name }}!
{{if .ShowMessage}}
Message: {{.Message}}
{{else}}
No message for you.
{{end}}
{{- end -}}
`,
			},
			want: []Struct{
				{
					Name: "Sample",
					Fields: []StructField{
						{Name: "Name", Type: "string"},
						{Name: "ShowMessage", Type: "bool"},
						{Name: "Message", Type: "string"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "3. with if-else expressions",
			args: args{
				tmpl: /*gotmpl*/ `
{{ define "Sample" }}
{{- /* @param Name string */}}
{{- /* @param Message string */}}
{{- /* @param ShowMessage bool */}}
{{- /* @param Greeting string */}}
Hello, {{ .Name }}!
{{if .ShowMessage}}
Message: {{.Message}}
{{else}}
Greeting: {{.Greeting}}
No message for you.
{{end}}
{{end}}
`,
			},
			want: []Struct{
				{
					Name: "Sample",
					Fields: []StructField{
						{Name: "Name", Type: "string"},
						{Name: "ShowMessage", Type: "bool"},
						{Name: "Message", Type: "string"},
						{Name: "Greeting", Type: "string"},
					},
				},
			},
			wantErr: false,
		},
		// write new cases and complex cases
		{
			name: "4. with if-else with complex conditional",
			args: args{
				tmpl: /*gotmpl*/ `
{{ define "Sample" }}
{{- /* @param Name string */}}
{{- /* @param Message string */}}
{{- /* @param ShowMessage bool */}}
{{- /* @param Greeting string */}}
Hello, {{ .Name }}!
{{if (gt (len .Message) 0 )}}
Message: {{.Message}}
{{else}}
Greeting: {{.Greeting}}
{{end}}
{{end}}
`,
			},
			want: []Struct{
				{
					Name: "Sample",
					Fields: []StructField{
						{Name: "Name", Type: "string"},
						{Name: "Message", Type: "string"},
						{Name: "Greeting", Type: "string"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "5. with if-else with complex conditional, with a missing type annotation",
			args: args{
				tmpl: /*gotmpl*/ `
{{ define "Sample" }}
{{- /* @param Name string */}}
{{- /* @param Message string */}}
Hello, {{ .Name }}!
{{if (gt (len .Message) 0 )}}
Message: {{.Message}}
{{else}}
Greeting: {{.Greeting}}
{{end}}
{{end}}
`,
			},
			want: []Struct{
				{
					Name: "Sample",
					Fields: []StructField{
						{Name: "Name", Type: "string"},
						{Name: "Message", Type: "string"},
						{Name: "Greeting", Type: "any"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "6. with range block",
			args: args{
				tmpl: /*gotmpl*/ `
		{{ define "Sample" }}
{{- /* @param names []string */}}
		{{- range $item := .names }}
		Hello, {{ $item }}!
		{{- end }}
		{{- end }}
		`,
			},
			want: []Struct{
				{
					Name: "Sample",
					Fields: []StructField{
						{Name: "Names", Type: "[]string"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "7. if block with go template inbuilt functions",
			args: args{
				tmpl: /*gotmpl*/ `
					{{ define "Sample" }}
					{{- /* @param name string */}}
					{{- if eq "sample" .name }}
					Hello SAMPLE, You Matched!
					{{- end }}
					{{- end }}
		`,
			},
			want: []Struct{
				{
					Name: "Sample",
					Fields: []StructField{
						{Name: "Name", Type: "string"},
					},
				},
			},
			wantErr: false,
		},
	}
	for _idx, tt := range tests {
		idx := _idx + 1

		// if idx != 1 {
		// 	return
		// }

		t.Run(tt.name, func(t *testing.T) {
			fp, err := NewFileParser(tt.args.tmpl, defaultStructName)
			if err != nil {
				panic(err)
			}

			_, _, structs, err := fp.Parse()

			// _, gotStructs, err := generateStructs(tt.args.tmpl)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got := ""
			for i := range structs {
				s, err := structs[i].String()
				if err != nil {
					t.Errorf("generateStruct() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				got += s
				got += "\n"
			}

			want := ""
			for i := range tt.want {
				s, err := tt.want[i].String()
				if err != nil {
					t.Errorf("generateStruct() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				want += s
				want += "\n"
			}

			if got != want {
				t.Errorf("got != want")

				tmpdir := filepath.Join(os.TempDir(), "struct-gen", fmt.Sprintf("test-%d", idx))
				if err := os.MkdirAll(tmpdir, 0o755); err != nil {
					panic(err)
				}

				os.WriteFile(filepath.Join(tmpdir, "got.txt"), []byte(got), 0o644)
				os.WriteFile(filepath.Join(tmpdir, "want.txt"), []byte(want), 0o644)

				ncmd := func(file string) {
					t.Logf("%s\n", file)
					cmd := exec.Command("bat", filepath.Join(tmpdir, file))
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Run()
				}

				ncmd("got.txt")
				ncmd("want.txt")
			}

			// cmd := exec.Command("diff", filepath.Join(tmpdir, "got.txt"), filepath.Join(tmpdir, "want.txt"))
			// cmd.Stdout = os.Stdout
			// cmd.Stderr = os.Stderr
			// if err := cmd.Run(); err != nil {
			// 	t.Errorf("generateStruct() error = %v, wantErr %v", err, tt.wantErr)
			// 	return
			// }
		})
	}
}
