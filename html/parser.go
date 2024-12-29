package html

import (
	"bytes"
	"html/template"
	"io"
	"log/slog"
	"regexp"

	"github.com/nxtcoder17/go-template/pkg/log"
	"golang.org/x/net/html"
)

var logger *slog.Logger

func init() {
	l := log.New(log.ShowCallerInfo())
	logger = l.Slog()
}

func copyChildren(oldNode, newNode *html.Node) {
	logger.Debug("HERE", "oldnode", oldNode.Data, "new node", newNode.Data)
	for c := oldNode.FirstChild; c != nil; c = c.NextSibling {
		logger.Debug("HERE", "oldnode.FirstChild", c == nil, "new node", newNode == nil)
		nc := html.Node{
			FirstChild: c.FirstChild,
			LastChild:  c.LastChild,
			Type:       c.Type,
			DataAtom:   c.DataAtom,
			Data:       c.Data,
			Namespace:  c.Namespace,
			Attr:       c.Attr,
		}

		newNode.AppendChild(&nc)
	}
}

// replaceNode replaces a node in the DOM
func replaceNode(oldNode, newNode *html.Node) {
	parent := oldNode.Parent
	if parent == nil {
		return // Cannot replace if no parent exists
	}

	// INFO: need to remove otherwise appendchild panics
	newNode.Parent = nil
	newNode.PrevSibling = nil
	newNode.NextSibling = nil

	switch newNode.Data {
	case "fragment":
		{
			logger.Info("parent to fragment is", "node", parent.Data)
			copyChildren(newNode, parent)
		}
	default:
		{
			parent.InsertBefore(newNode, oldNode)
		}
	}

	// Insert the new node and remove the old node
	parent.RemoveChild(oldNode)
}

func findHeadElement(n *html.Node) *html.Node {
	if n.Type == html.ElementNode {
		logger.Debug("find HEAD element", "type", n.Type, "data", n.Data, "attr", n.Attr, "data-atom", n.DataAtom)
		switch n.Data {
		case "head":
			return n
		}
	}
	// Recursively process child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if n := findHeadElement(c); n != nil {
			return n
		}
	}

	return nil
}

func findComponentBody(n *html.Node) *html.Node {
	logger.Debug("find BODY child", "type", n.Type, "data", n.Data, "attr", n.Attr, "data-atom", n.DataAtom)
	if n.Type == html.ElementNode {
		switch n.Data {
		case "body":
			return n.FirstChild
		case "head":
			{
				if n.FirstChild != nil {
					return n
				}
			}
		}
	}
	// Recursively process child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if n := findComponentBody(c); n != nil {
			return n
		}
	}

	return nil
}

func parseHTML(n *html.Node, onTargetNodeFound func(node *html.Node)) error {
	if n.Type == html.ElementNode && !HTMLTags.Has(n.Data) {
		logNode("target-node", n)
		onTargetNodeFound(n)
	}

	// Recursively process child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parseHTML(c, onTargetNodeFound)
	}

	return nil
}

func renderHTML(w io.Writer, n *html.Node) error {
	logger.Debug("RENDERING html")
	return html.Render(w, n)
}

func parseComponentHTML(reader io.Reader) (*html.Node, error) {
	n, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}

	return findComponentBody(n), err
}

func htmlAttrsToMap(attrs []html.Attribute) map[string]any {
	m := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		m[attr.Key] = attr.Val
	}

	return m
}

type Params struct {
	Input        io.Reader
	Output       io.Writer
	Template     *template.Template
	GetComponent func(name string, attrs map[string]any) (Component, error)
}

var re = regexp.MustCompile(`<([A-Za-z0-9]+)([^>]*)\/>`)

func fixSelfClosingTags(r io.Reader) (io.Reader, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	b2 := re.ReplaceAll(b, []byte(`<$1$2></$1>`))
	return bytes.NewReader(b2), nil
}

func Parse(p Params) error {
	r, err := fixSelfClosingTags(p.Input)
	if err != nil {
		return err
	}

	n, err := html.Parse(r)
	if err != nil {
		panic(err)
	}

	headEl := findHeadElement(n)

	var replaceNodes []*html.Node
	onTargetNodeFound := func(n *html.Node) {
		replaceNodes = append(replaceNodes, n)
	}

	parseHTML(n, onTargetNodeFound)

	for _, rn := range replaceNodes {
		if rn.Type == html.ElementNode {
			component, err := p.GetComponent(rn.Data, htmlAttrsToMap(rn.Attr))
			if err != nil {
				return err
			}

			b := new(bytes.Buffer)

			if err := component.Render(p.Template, b); err != nil {
				return err
			}

			newNode, err := parseComponentHTML(b)
			if err != nil {
				return err
			}

			switch newNode.Data {
			case "head":
				{
					if headEl == nil {
						headEl = rn
					}

					// printChild(rn)

					copyChildren(newNode, headEl)
					parent := rn.Parent
					parent.RemoveChild(rn)
				}
			default:
				{
					copyChildren(rn, newNode)
					replaceNode(rn, newNode)
				}
			}
		}
	}

	return renderHTML(p.Output, n)
}
