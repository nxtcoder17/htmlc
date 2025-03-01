package template

import (
	"bytes"
	_ "embed"
	"go/format"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type printOutputArgs struct {
	Package                 string
	Imports                 []string
	Structs                 []Struct
	ParseFuncName           string
	InputTemplate           string
	GeneratingForComponents bool
}

func (p *Parser) PrintParsedStructFile(writer io.Writer, args printOutputArgs) error {
	b := new(bytes.Buffer)
	if err := p.parsedStructFileTemplate.Execute(b, args); err != nil {
		return err
	}

	formatted, err := format.Source(b.Bytes())
	if err != nil {
		slog.Error("while formatting generated parsed struct file", "err", err)
		_, err := writer.Write(b.Bytes())
		return err
	}

	_, err = writer.Write(formatted)
	return err
}

type PrintPkgInitFileArgs struct {
	Dir                     string
	Package                 string
	GeneratingForComponents bool
}

func (p *Parser) PrintPkgInitFile(args PrintPkgInitFileArgs) error {
	w, err := os.Create(filepath.Join(args.Dir, "init.go"))
	if err != nil {
		return err
	}

	b := new(bytes.Buffer)
	if err := p.parsedPkgInitFileTemplate.Execute(b, map[string]any{
		"Package":                 args.Package,
		"TemplateImport":          p.templateImport,
		"GeneratingForComponents": args.GeneratingForComponents,
	}); err != nil {
		return err
	}

	formatted, err := format.Source(b.Bytes())
	if err != nil {
		slog.Error("while formatting generated pkg init file", "err", err)
		_, err := w.Write(b.Bytes())
		return err
	}

	_, err = w.Write(formatted)
	return err
}
