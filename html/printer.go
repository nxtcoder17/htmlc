package html

import (
	"io"

	"golang.org/x/net/html"
)

func renderHTML(w io.Writer, n *html.Node) error {
	logger.Debug("RENDERING html")
	return html.Render(w, n)
}
