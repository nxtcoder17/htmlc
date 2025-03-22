package template

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"

	fn "github.com/nxtcoder17/htmlc/pkg/functions"
)

var Logger *slog.Logger

func init() {
	Logger = slog.Default()
}

const paramLabel string = "__param__"

var defaultStructName = "YourStdoutStruct"

func parseNode(p parse.Node, prefix string, onNodeFound func(sf StructField, isComment bool)) {
	switch node := p.(type) {
	case *parse.IdentifierNode:
		{
			onNodeFound(toStructField(node.String(), "any"), false)
			Logger.Debug("identifier node", "prefix", prefix, "node", node.String())
		}

	// CASE: .Variable
	case *parse.FieldNode:
		{
			for _, v := range node.Ident {
				onNodeFound(toStructField(v, "any"), false)
			}
			Logger.Debug("field node", "prefix", prefix, "node", node.String())
		}

	// CASE: {{.Variable}}
	case *parse.ActionNode:
		{
			Logger.Debug("action node", "prefix", prefix, "node", node.Pipe.String())
			for _, c := range node.Pipe.Cmds {
				Logger.Debug("got comment", "c.Args", c.Args)

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
					Logger.Debug(fmt.Sprintf("cmds[%d]", idx), "type", ci.Type(), "cmd", ci.String())
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
			Logger.Debug("if node", "prefix", prefix, "node", node.String(), "pipe", node.Pipe.Cmds)

			for i := range node.Pipe.Cmds {
				parseNode(node.Pipe.Cmds[i], prefix, onNodeFound)
			}

			parseNode(node.List, "if", onNodeFound)
			parseNode(node.ElseList, "else", onNodeFound)
		}
	case *parse.ListNode:
		Logger.Debug("list node", "prefix", prefix, "node", node.String())
		for i := range node.Nodes {
			parseNode(node.Nodes[i], fmt.Sprintf("list [%d]", i), onNodeFound)
		}

	case *parse.CommandNode:
		Logger.Debug("command-node", "node", node.String())
		for i := range node.Args {
			parseNode(node.Args[i], fmt.Sprintf("list [%d]", i), onNodeFound)
		}
	case *parse.RangeNode:
		Logger.Debug("range-node", "node", node.String(), "struct", fmt.Sprintf("%+v", *node.Pipe))
		for i := range node.Pipe.Cmds {
			parseNode(node.Pipe.Cmds[i], "range-node", onNodeFound)
		}
	}
}

func structFromTemplate(structName string, t *template.Template) (Struct, error) {
	var fields []StructField
	commentsMap := make(map[string]StructField)

	fieldsMap := make(map[string]int)

	onVarFound := func(sf StructField, isFromComment bool) {
		if isFromComment {
			if _, ok := commentsMap[sf.Name]; !ok {
				commentsMap[sf.Name] = sf
			}
			return
		}

		if sf.Name == "Props" || sf.Name == "Remaining" {
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

	for i := range fields {
		if sf, ok := commentsMap[fields[i].Name]; ok {
			if idx := strings.LastIndex(sf.Type, "."); idx != -1 {
				pkg := sf.Type[:idx]
				imports = append(imports, pkg)
				fields[i].Package = &pkg
				fields[i].Type = fmt.Sprintf("%s.%s", filepath.Base(pkg), sf.Type[idx+1:])
				fields[i].Tag = sf.Tag
				continue
			}
			fields[i].Type = sf.Type
			fields[i].Tag = sf.Tag
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
	return re.ReplaceAllString(tmpl, fmt.Sprintf(`{{- %s "$1" "$2" -}}`, paramLabel))
}

func removeParamComments(tmpl string) string {
	return re.ReplaceAllString(tmpl, "")
}

//go:embed printer-parsed-struct.go.tpl
var ParsedStructOutputTemplate string

//go:embed printer-pkg-init.go.tpl
var ParsedPkgInitFileTemplate string

type Parser struct {
	templateImport string

	parsedStructFileTemplate  *template.Template
	parsedPkgInitFileTemplate *template.Template
}

type ParseOptions struct {
	GlobPatterns            []string
	StructNamePrefix        *string
	PreProcess              func(tmpl string) (string, error)
	GeneratingForComponents bool
}

func (p *Parser) ParseDir(inputDir string, outputDir string, outputPkg string, opts ...ParseOptions) error {
	opt := ParseOptions{}

	if len(opts) >= 1 {
		opt = opts[0]
	}

	if opt.GlobPatterns == nil {
		opt.GlobPatterns = []string{"*.html"}
	}

	if err := os.MkdirAll(outputDir, 0o766); err != nil {
		return err
	}

	listings, err := fn.RecursiveLs(inputDir, opt.GlobPatterns)
	if err != nil {
		return err
	}

	// if len(listings) == 0 {
	// 	return fmt.Errorf("template: pattern matches no files: %#q", opt.GlobPatterns)
	// }

	if err := p.PrintPkgInitFile(PrintPkgInitFileArgs{
		Dir:                     outputDir,
		Package:                 outputPkg,
		GeneratingForComponents: opt.GeneratingForComponents,
	}); err != nil {
		return err
	}

	for _, item := range listings {
		Logger.Debug("template-parser | listings", "item", item)

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

		if err := p.parse(string(input), defStructName, parseFuncName, &outFile, outputPkg, opt); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) Parse(input string, outputFile *string, outputPkg string, opts ParseOptions) error {
	parseFuncName := "parseStdout"
	return p.parse(input, defaultStructName, parseFuncName, outputFile, outputPkg, opts)
}

func (p *Parser) parse(input string, structName string, parseFuncName string, outputFile *string, outputPkg string, opts ParseOptions) error {
	fp, err := NewFileParser(string(input), structName)
	if err != nil {
		return err
	}

	tmpl, imports, structs, err := fp.Parse()
	if err != nil {
		return err
	}

	// INFO: to remove @param comments, in generated file
	// tmpl = removeParamComments(tmpl)

	imports = append(imports, p.templateImport)
	out := os.Stdout

	if outputFile != nil {
		out, err = os.Create(*outputFile)
		if err != nil {
			return err
		}
	}

	return p.PrintParsedStructFile(out, printOutputArgs{
		Package:                 outputPkg,
		Imports:                 imports,
		Structs:                 structs,
		ParseFuncName:           parseFuncName,
		InputTemplate:           tmpl,
		GeneratingForComponents: opts.GeneratingForComponents,
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

		"lowercase": func(str string) string {
			return strings.ToLower(str)
		},
	}

	t, err := template.New("parse").Funcs(funcs).Parse(ParsedStructOutputTemplate)
	if err != nil {
		return nil, err
	}

	outputPkgTmpl, err := template.New("parse").Funcs(funcs).Parse(ParsedPkgInitFileTemplate)
	if err != nil {
		return nil, err
	}

	return &Parser{
		templateImport:            fmt.Sprintf("%s/template", ttype),
		parsedStructFileTemplate:  t,
		parsedPkgInitFileTemplate: outputPkgTmpl,
		// allTemplates:   template.New("all-templates").Funcs(funcs),
	}, nil
}
