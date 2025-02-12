package template

import (
	"bytes"
	_ "embed"
	"go/format"
	"io"
	"os"
	"path/filepath"
)

type printOutputArgs struct {
	Package       string
	Imports       []string
	Structs       []Struct
	ParseFuncName string
	InputTemplate string
}

func (p *Parser) PrintOutputFile(writer io.Writer, args printOutputArgs) error {
	b := new(bytes.Buffer)
	if err := p.outputFileTmpl.Execute(b, args); err != nil {
		return err
	}

  var err error
	_, err = writer.Write(b.Bytes())
	return err

	// formatted, err := format.Source(b.Bytes())
	// if err != nil {
	// 	return err
	// }
	//
	// _, err = writer.Write(formatted)
	// return err
}

func (p *Parser) PrintOutputPkgFile(dir string, pkgName string) error {
	w, err := os.Create(filepath.Join(dir, "generated.go"))
	if err != nil {
		return err
	}

	b := new(bytes.Buffer)
	p.outputPkgTmpl.Execute(w, map[string]any{
		"package":         pkgName,
		"template_import": p.templateImport,
	})

	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return err
	}

	_, err = w.Write(formatted)
	return err
}
