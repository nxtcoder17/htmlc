package html

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func nodePrinter(n *html.Node, withChildren bool) (string, []string) {
	if n == nil {
		panic("called with nil node")
	}

	attrs := make([]string, 0, len(n.Attr))
	for _, attr := range n.Attr {
		attrs = append(attrs, fmt.Sprintf("%s=%s", attr.Key, attr.Val))
	}

	children := getFilteredChildren(n)
	childrenMsg := make([]string, 0, len(children))
	if withChildren {
		for _, child := range children {
			m, _ := nodePrinter(child, false)
			childrenMsg = append(childrenMsg, m)
		}
	}

	msg := fmt.Sprintf("<%s %s> (%d) (%d children)", strings.ToUpper(n.Data), attrs, n.DataAtom, len(children))
	return msg, childrenMsg
}

func logNode(msg string, n *html.Node) {
	s, c := nodePrinter(n, true)
	logger.Info(msg, "element", s, "children", c)
}
