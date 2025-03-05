package html

import (
	"io"

	"golang.org/x/net/html"
)

func renderHTML(w io.Writer, n *html.Node) error {
	if n == nil {
		return nil
	}
	logger.Debug("RENDERING html")
	return html.Render(w, n)
}
