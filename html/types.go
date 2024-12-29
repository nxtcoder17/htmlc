package html

import (
	"html/template"
	"io"
)

type Component interface {
	Render(*template.Template, io.Writer) error
}
