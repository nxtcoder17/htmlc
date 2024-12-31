package template

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"
)

func pp(prefix string, v ...any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	_ = b
	_ = prefix
	// fmt.Printf("%s:\t%s\n", prefix, b)
}

func pp2(prefix string, v ...any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	_, _ = b, prefix
	// fmt.Printf("%s:\t%s\n", prefix, b)
}

const paramLabel string = "__param__"

func parseNode(p parse.Node, prefix string, onNodeFound func(sf StructField, isComment bool)) {
	// fmt.Println("got node", prefix, p.String(), p.Type())
	switch node := p.(type) {
	case *parse.IdentifierNode:
		{
			onNodeFound(toStructField(node.String(), "any"), false)
			// onNodeFound(node.String(), "any", false)
			pp(fmt.Sprintf("identifier node (%s)", prefix), node.String())
		}

	// CASE: .Variable
	case *parse.FieldNode:
		{
			for _, v := range node.Ident {
				onNodeFound(toStructField(v, "any"), false)
			}
			pp(fmt.Sprintf("field node (%s)", prefix), node.Type(), node.String())
		}

	// CASE: {{.Variable}}
	case *parse.ActionNode:
		{
			pp2(fmt.Sprintf("action node (%s)", prefix), node.Pipe.String())
			for _, c := range node.Pipe.Cmds {
				pp2("cmds", c.String())

				if len(c.Args) >= 3 && c.Args[0].String() == paramLabel {
					// INFO: it is our param comment
					varName, err := strconv.Unquote(c.Args[1].String())
					if err != nil {
						continue
					}

					varType, err := strconv.Unquote(c.Args[2].String())
					if err != nil {
						continue
					}

					onNodeFound(toStructField(varName, varType), true)
					continue
				}

				for idx, ci := range c.Args {
					pp2(fmt.Sprintf("cmds[%d]", idx), ci.Type(), ci.String())
					parseNode(ci, fmt.Sprintf("cmds[%d]", idx), onNodeFound)
				}
			}
			// for _, d := range node.Pipe.Decl {
			// 	pp("decl", d.String())
			// }
			// *result = append(*result, node.Pipe.String())
			// return
		}
	case *parse.IfNode:
		{
			pp(fmt.Sprintf("if node (%s)", prefix), node.String())

			pp("if node pipe", node.Pipe.Cmds)
			for i := range node.Pipe.Cmds {
				parseNode(node.Pipe.Cmds[i], prefix, onNodeFound)
			}

			parseNode(node.List, "if", onNodeFound)
			parseNode(node.ElseList, "else", onNodeFound)
		}
	case *parse.ListNode:
		pp(fmt.Sprintf("list node (%s)", prefix), node.String())
		for i := range node.Nodes {
			parseNode(node.Nodes[i], fmt.Sprintf("list [%d]", i), onNodeFound)
		}

	case *parse.CommandNode:
		fmt.Println("command-node", node.String())
		for i := range node.Args {
			parseNode(node.Args[i], fmt.Sprintf("list [%d]", i), onNodeFound)
		}

	case *parse.CommentNode:
		pp(fmt.Sprintf("comment node (%s)", prefix), node.Text)
	}
}

func structFromTemplate(structName string, t *template.Template) (Struct, error) {
	var fields []StructField
	commentsMap := make(map[string]string)

	fieldsMap := make(map[string]int)

	onVarFound := func(sf StructField, isFromComment bool) {
		if isFromComment {
			if _, ok := commentsMap[sf.Name]; !ok {
				commentsMap[sf.Name] = sf.Type
			}
			return
		}

		if _, ok := fieldsMap[sf.Name]; !ok {
			fields = append(fields, sf)
			fieldsMap[sf.Name] = len(fields) - 1
		}
	}

	for _, n := range t.Root.Nodes {
		parseNode(n, "", onVarFound)
	}

	var imports []string

	// fmt.Printf("fields: %+v\n", fields)
	// fmt.Printf("commentsMap: %+v\n", commentsMap)

	for i := range fields {
		if commentType, ok := commentsMap[fields[i].Name]; ok {
			if idx := strings.LastIndex(commentType, "."); idx != -1 {
				pkg := commentType[:idx]
				imports = append(imports, pkg)
				fields[i].Package = &pkg
				fields[i].Type = fmt.Sprintf("%s.%s", filepath.Base(pkg), commentType[idx+1:])
				continue
			}
			fields[i].Type = commentType
		}
	}

	return Struct{
		Name:    structName,
		Fields:  fields,
		Imports: imports,
	}, nil
}

var re = regexp.MustCompile(`{{-?\s*/\*\s* @param \s*(.*) \s*(.*) ()\*/}}`)

func fixParamComments(tmpl string) string {
	// matches := re.FindAllStringSubmatch(tmpl, -1)
	// for _, m := range matches {
	// 	fmt.Printf("item (%s, %s)\n", m[1], m[2])
	// }

	return re.ReplaceAllString(tmpl, fmt.Sprintf(`{{- %s "$1" "$2" -}}`, paramLabel))
}

func parseTemplateString(input string, defaultStructName string) (*template.Template, string, error) {
	// Parse the template
	t := template.New("t:parser")
	t.Funcs(template.FuncMap{
		paramLabel: func(key, value string) string {
			return "/* comment */"
		},
	})

	t, err := t.Parse(fixParamComments(input))
	if err != nil {
		return nil, "", nil
	}

	if len(t.Templates()) == 1 {
		fmt.Println("HERE................")
		return parseTemplateString(strings.Join([]string{
			fmt.Sprintf(`{{- define "%s" }}`, defaultStructName),
			input,
			`{{- end }}`,
		}, "\n"), defaultStructName)
	}

	return t, input, nil
}

func generateStructs(tmpl string, defaultStructName string) (fixedTemplate string, imports []string, structs []Struct, err error) {
	imports = append(imports,
		"github.com/go-playground/validator/v10",
		"io",
		"encoding/json",
	)

	t, tmpl, err := parseTemplateString(tmpl, defaultStructName)

	result := make([]Struct, 0, len(t.Templates()))
	for _, v := range t.Templates() {
		if v.Name() == "t:parser" && len(t.Templates()) > 1 {
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

	return tmpl, imports, result, nil
}

// type Option struct {
// 	TemplateType string
// 	Input        io.Reader
// 	Output       io.Writer
// 	OutPkg       string
// 	OutputDir    *string
// }
//
// func Parse(opt Option) error {
// 	if opt.Input == nil {
// 		return fmt.Errorf("must specify option.Input")
// 	}
//
// 	if opt.OutPkg == "" {
// 		return fmt.Errorf("must specify option.OutPkg")
// 	}
//
// 	if opt.Output == nil {
// 		return fmt.Errorf("must specify option.Output")
// 	}
//
// 	b, err := io.ReadAll(opt.Input)
// 	if err != nil {
// 		return err
// 	}
//
// 	imports, structs, err := generateStructs(string(b))
// 	if err != nil {
// 		return err
// 	}
//
// 	if opt.Output == nil {
// 		opt.Output = os.Stdout
// 	}
//
// 	if opt.OutputDir != nil {
// 		printPackageLevelFile(*opt.OutputDir, opt.OutPkg, "html")
// 	}
//
// 	if err := printOutput(opt.Output, opt.OutPkg, imports, structs); err != nil {
// 		return err
// 	}
//
// 	return nil
// }

//go:embed printer_template.go.tpl
var outputTemplate string

//go:embed generated_template.go.tpl
var outputPkgTemplate string

type Parser struct {
	templateImport string

	outputFileTmpl *template.Template
	outputPkgTmpl  *template.Template

	// allTemplates *template.Template
	// postProcess  func(t *template.Template) (string, error)
}

type ParseOptions struct {
	GlobPatterns     []string
	StructNamePrefix *string
}

func (p *Parser) ParseDir(inputDir string, outputDir string, outputPkg string, opts ...ParseOptions) error {
	opt := ParseOptions{}

	if len(opts) >= 1 {
		opt = opts[0]
	}

	if opt.GlobPatterns == nil {
		opt.GlobPatterns = []string{"*.html", "**/*.html"}
	}

	fmt.Println("pre-processing", "output-pkg", outputPkg)

	var listings []string

	for _, pattern := range opt.GlobPatterns {
		list, err := fs.Glob(os.DirFS(inputDir), pattern)
		if err != nil {
			panic(err)
		}
		listings = append(listings, list...)
	}

	if err := os.MkdirAll(outputDir, 0o766); err != nil {
		panic(err)
	}

	if len(listings) == 0 {
		panic(fmt.Errorf("template: pattern matches no files: %#q", opt.GlobPatterns))
	}

	if err := p.PrintOutputPkgFile(outputDir, outputPkg); err != nil {
		return err
	}

	for _, item := range listings {
		fmt.Printf("template-parser | listing | %s\n", item)

		input, err := os.ReadFile(filepath.Join(inputDir, item))
		if err != nil {
			return err
		}

		base := filepath.Base(item)
		base = toFieldName(base[:len(base)-len(filepath.Ext(base))])

		defStructName := base
		if opt.StructNamePrefix != nil {
			defStructName = toFieldName(*opt.StructNamePrefix + base)
		}

		parseFuncName := "parse" + defStructName

		outFile := filepath.Join(outputDir, fmt.Sprintf("%s_generated.go", item))
		if err := os.MkdirAll(filepath.Dir(outFile), 0o766); err != nil {
			return err
		}

		if err := p.parse(string(input), defStructName, parseFuncName, &outFile, outputPkg); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) Parse(input string, outputFile *string, outputPkg string) error {
	parseFuncName := "parseStdout"
	return p.parse(input, "YourStdoutStruct", parseFuncName, outputFile, outputPkg)
}

func (p *Parser) parse(input string, structName string, parseFuncName string, outputFile *string, outputPkg string) error {
	// tmpl, imports, structs, err := generateStructs(input, structName)
	// if err != nil {
	// 	return err
	// }

	fp, err := NewFileParser(string(input), structName)
	if err != nil {
		return err
	}

	tmpl, imports, structs, err := fp.Parse()

	// if _, err := p.allTemplates.Parse(tmpl); err != nil {
	// 	return err
	// }

	imports = append(imports, p.templateImport)
	out := os.Stdout

	if outputFile != nil {
		out, err = os.Create(*outputFile)
		if err != nil {
			return err
		}
		// outDir := filepath.Dir(*outputFile)
		// if err := p.PrintOutputPkgFile(outDir, outputPkg); err != nil {
		// 	return err
		// }
	}

	fmt.Println("HELLO .......", outputPkg)

	return p.PrintOutputFile(out, printOutputArgs{
		Package:       outputPkg,
		Imports:       imports,
		Structs:       structs,
		ParseFuncName: parseFuncName,
		InputTemplate: tmpl,
	})
}

type TemplateType string

const (
	Html TemplateType = "html"
	Text TemplateType = "text"
)

func NewParser(ttype TemplateType) (*Parser, error) {
	if ttype != Html && ttype != Text {
		return nil, fmt.Errorf("unsupported template type (%s), only text, and html are supported", ttype)
	}

	funcs := template.FuncMap{
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
	}

	t, err := template.New("parse").Funcs(funcs).Parse(outputTemplate)
	if err != nil {
		return nil, err
	}

	outputPkgTmpl, err := template.New("parse").Funcs(funcs).Parse(outputPkgTemplate)
	if err != nil {
		return nil, err
	}

	return &Parser{
		templateImport: fmt.Sprintf("%s/template", ttype),
		outputFileTmpl: t,
		outputPkgTmpl:  outputPkgTmpl,
		// allTemplates:   template.New("all-templates").Funcs(funcs),
	}, nil
}
