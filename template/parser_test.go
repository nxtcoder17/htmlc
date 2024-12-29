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
					Name: "YourStructName",
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
	}
	for _idx, tt := range tests {
		idx := _idx + 1

		// if idx != 1 {
		// 	return
		// }

		t.Run(tt.name, func(t *testing.T) {
			_, gotStructs, err := generateStructs(tt.args.tmpl)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got := ""
			for i := range gotStructs {
				s, err := gotStructs[i].String()
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

			tmpdir := filepath.Join(os.TempDir(), "struct-gen", fmt.Sprintf("test-%d", idx))
			if err := os.MkdirAll(tmpdir, 0o755); err != nil {
				panic(err)
			}

			os.WriteFile(filepath.Join(tmpdir, "got.txt"), []byte(got), 0o644)
			os.WriteFile(filepath.Join(tmpdir, "want.txt"), []byte(want), 0o644)

			cmd := exec.Command("delta", filepath.Join(tmpdir, "got.txt"), filepath.Join(tmpdir, "want.txt"))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Errorf("generateStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
