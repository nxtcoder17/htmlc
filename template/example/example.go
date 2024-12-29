package main

import (
	"flag"
	"io"
	"os"

	"github.com/nxtcoder17/go-template/pkg/parser/template"
)

func main() {
	var in, out, ttype, outpkg string
	flag.StringVar(&in, "in", "", "--in <template>")
	flag.StringVar(&ttype, "type", "text", "--type <text|html>")
	flag.StringVar(&out, "out", "", "--out <template>")
	flag.StringVar(&outpkg, "out-pkg", "", "--out-pkg <output-package-name>")

	flag.Parse()

	if in == "" || outpkg == "" || ttype == "" {
		panic("must specify in, type and out-pkg")
	}

	var writer io.WriteCloser = os.Stdout
	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			panic(err)
		}
		writer = f
	}

	f, err := os.Open(in)
	if err != nil {
		panic(err)
	}

	if err := template.Parse(template.Option{
		TemplateType: ttype,
		Input:        f,
		Output:       writer,
		OutPkg:       outpkg,
	}); err != nil {
		panic(err)
	}
}
