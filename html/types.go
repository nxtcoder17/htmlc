package html

import (
	"io"
)

type Component interface {
	Render(io.Writer) error
}
