package html

import (
	"bytes"
	"html/template"
	"io"
	"log/slog"
	"regexp"

	"github.com/nxtcoder17/go-template/pkg/log"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var logger *slog.Logger

func init() {
	l := log.New(log.ShowCallerInfo())
	logger = l.Slog()
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

func parseHTML(n *html.Node, onTargetNodeFound func(node *html.Node)) error {
	if n.Data == "body" {
		logNode("body-node", n)
	}
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

func parseWithFragments(reader io.Reader) (*html.Node, error) {
	b, err := fixSelfClosingTags(reader)
	if err != nil {
		return nil, err
	}

	htmlNode := &html.Node{
		Type:     html.ElementNode,
		Data:     "html",
		DataAtom: atom.Html,
	}

	nl, err := html.ParseFragment(bytes.NewReader(b), htmlNode)
	if err != nil {
		return nil, err
	}

	if len(nl) != 2 {
		return nil, err
	}

	head, body := nl[0], nl[1]

	headChildren := getChildren(head)
	bodyChildren := getChildren(body)

	switch {
	case len(headChildren) > 0 && len(bodyChildren) > 0:
		return html.Parse(bytes.NewReader(b))
	case len(headChildren) > 0:
		return head, nil

	// INFO: len(bodyChildren) > 0

	case len(bodyChildren) == 1:
		return body.FirstChild, nil

	case len(bodyChildren) > 1:
		newNode := &html.Node{
			Type: html.ElementNode,
			Data: "div",
		}

		copyChildren(body, newNode)
		return newNode, nil

	default:
		return htmlNode, err
	}
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

func fixSelfClosingTags(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	b2 := re.ReplaceAll(b, []byte(`<$1$2></$1>`))
	return b2, nil
}

func Parse(p Params) error {
	b, err := fixSelfClosingTags(p.Input)
	if err != nil {
		return err
	}

	_ = atom.Body

	n, err := parseWithFragments(bytes.NewReader(b))
	if err != nil {
		return err
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

			// newNode, err := parseComponentHTML(b)
			newNode, err := parseWithFragments(b)
			if err != nil {
				return err
			}

			switch newNode.Data {
			case "head":
				{
					if headEl == nil {
						headEl = rn
					}

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
