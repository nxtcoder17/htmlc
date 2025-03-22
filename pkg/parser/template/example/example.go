package main

import (
	"flag"

	"github.com/nxtcoder17/htmlc/pkg/parser/template"
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

	p, err := template.NewParser(template.TemplateType(ttype))
	if err != nil {
		panic(err)
	}

	if err := p.Parse(in, nil, outpkg, template.ParseOptions{}); err != nil {
		panic(err)
	}
}
