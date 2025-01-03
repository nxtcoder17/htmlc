package html

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"testing"

	"golang.org/x/net/html"
)

func Test_parseHTML(t *testing.T) {
	type args struct {
		n                 *html.Node
		onTargetNodeFound func(node *html.Node)
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := parseHTML(tt.args.n, tt.args.onTargetNodeFound); (err != nil) != tt.wantErr {
				t.Errorf("parseHTML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_parseWithFragments(t *testing.T) {
	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *html.Node
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWithFragments(tt.args.reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWithFragments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseWithFragments() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	type args struct {
		p Params
	}
	tests := []struct {
		name string
		args args

		wantOutput []byte
		wantErr    bool
	}{
		{
			name: "1. parsing a simple no child element",
			args: args{
				p: Params{
					Input:    bytes.NewReader([]byte(`<input type="email"/>`)),
					Output:   new(bytes.Buffer),
					Template: nil,
					GetComponent: func(name string, attrs map[string]any) (Component, error) {
						return nil, fmt.Errorf("component not found")
					},
				},
			},
			wantOutput: []byte(`<input type="email"/>`),
			wantErr:    false,
		},

		{
			name: "2. parsing a simple template define block",
			args: args{
				p: Params{
					Input:    bytes.NewReader([]byte(`{{- define "Sample"}} <input type="email"/> {{- end }}`)),
					Output:   new(bytes.Buffer),
					Template: nil,
					GetComponent: func(name string, attrs map[string]any) (Component, error) {
						return nil, fmt.Errorf("component not found")
					},
				},
			},
			wantOutput: []byte(`<input type="email"/>`),
			wantErr:    false,
		},

		{
			name: "3. parsing a simple template define block, but with typed-templates",
			args: args{
				p: Params{
					Input: bytes.NewReader([]byte(`{{- define "Sample"}}
{{- /* @param label string */}}
<input type="email" label={{.label}} /> {{- end }}`)),
					Template: nil,
					GetComponent: func(name string, attrs map[string]any) (Component, error) {
						return nil, fmt.Errorf("component not found")
					},
				},
			},
			wantOutput: []byte(`<input type="email" label="{{.label}}"/>`),
			wantErr:    false,
		},

		{
			name: "4. parsing a simple template define block",
			args: args{
				p: Params{
					Input: bytes.NewReader([]byte(`{{- define "component/Input" }}
{{- /* @param class? string */}}
<div class="{{.class}} flex flex-row gap-8">
  {{- /* @param id? string */}}
  {{- /* @param label string */}}
  <label for="{{.id}}">{{.label}}</label>
  {{- /* @param type string */}}
  <input class="border-2 rounded-lg border-red-400" id="{{.id}}" type="{{.type}}" />
</div>
{{- end }}`)),
					Template: nil,
					GetComponent: func(name string, attrs map[string]any) (Component, error) {
						return nil, fmt.Errorf("component not found")
					},
				},
			},
			wantOutput: []byte(`
<div class="{{.class}} flex flex-row gap-8">
  <label for="{{.id}}">{{.label}}</label>
  <input class="border-2 rounded-lg border-red-400" id="{{.id}}" type="{{.type}}"/>
</div>
`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			tt.args.p.Output = out

			if err := Parse(tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			b, err := io.ReadAll(out)
			if err != nil {
				t.Errorf("failed to read from output stream")
			}

			// t.Logf("output:\n\tgot: %s\n\twant: %s\n", b, tt.wantOutput)

			wantOut := bytes.TrimSpace(tt.wantOutput)
			gotOut := bytes.TrimSpace(b)

			if !bytes.Equal(wantOut, gotOut) {
				t.Errorf("output did not match:\n\n\twant: %s\n\tgot: %s\n\n", wantOut, gotOut)
			}
		})
	}
}
