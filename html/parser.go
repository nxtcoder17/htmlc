package html

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"regexp"
	textTemplate "text/template"

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

	t := textTemplate.New("t:html:parser")
	t = t.Funcs(template.FuncMap{
		"__param__": func(k, v string) string {
			return "/* comment */"
		},
	})
	if _, err := t.Parse(string(b)); err != nil {
		return nil, err
	}

	if len(t.Templates()) > 1 {
		logger.Warn("HERE")
		content := ""
		for _, mt := range t.Templates() {
			if mt.Name() != t.Name() {
				content = mt.Root.String()
				break
			}
		}

		return parseWithFragments(bytes.NewReader([]byte(content)))
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

	// FIXME: bug in this flow
	// logNode("html", nl[0])
	logNode("head", head)
	logNode("body", body)

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
		//
		// default:
		// 	return htmlNode, err
	}
	return nil, fmt.Errorf("failed to parse html node :):(")
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

func parseHTMLAndTranspile(n *html.Node, t *template.Template, getComponent func(name string, attrs map[string]any) (Component, error)) (*html.Node, error) {
	var replaceNodes []*html.Node
	onTargetNodeFound := func(n *html.Node) {
		replaceNodes = append(replaceNodes, n)
	}

	if err := parseHTML(n, onTargetNodeFound); err != nil {
		return nil, err
	}

	headEl := findHeadElement(n)

	for _, rn := range replaceNodes {
		if rn.Type == html.ElementNode {
			component, err := getComponent(rn.Data, htmlAttrsToMap(rn.Attr))
			if err != nil {
				return nil, err
			}

			b := new(bytes.Buffer)

			if err := component.Render(t, b); err != nil {
				return nil, err
			}

			newNode, err := parseWithFragments(b)
			if err != nil {
				return nil, err
			}

			newNode, err = parseHTMLAndTranspile(newNode, t, getComponent)
			if err != nil {
				return nil, err
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

	return n, nil
}

func Parse(p Params) error {
	n, err := parseWithFragments(p.Input)
	// n, err := findEffectiveTree(p.Input)
	if err != nil {
		return err
	}

	n2, err := parseHTMLAndTranspile(n, p.Template, p.GetComponent)
	if err != nil {
		return err
	}

	return html.Render(p.Output, n2)

	//
	// headEl := findHeadElement(n)
	//
	// var replaceNodes []*html.Node
	// onTargetNodeFound := func(n *html.Node) {
	// 	replaceNodes = append(replaceNodes, n)
	// }
	//
	// parseHTML(n, onTargetNodeFound)
	//
	// for _, rn := range replaceNodes {
	// 	if rn.Type == html.ElementNode {
	// 		component, err := p.GetComponent(rn.Data, htmlAttrsToMap(rn.Attr))
	// 		if err != nil {
	// 			return err
	// 		}
	//
	// 		b := new(bytes.Buffer)
	//
	// 		if err := component.Render(p.Template, b); err != nil {
	// 			return err
	// 		}
	//
	// 		// newNode, err := parseComponentHTML(b)
	// 		newNode, err := parseWithFragments(b)
	// 		// newNode, err := findEffectiveTree(b)
	// 		if err != nil {
	// 			return err
	// 		}
	//
	// 		// parseHTML(newNode, onTargetNodeFound)
	//
	// 		switch newNode.Data {
	// 		case "head":
	// 			{
	// 				if headEl == nil {
	// 					headEl = rn
	// 				}
	//
	// 				copyChildren(newNode, headEl)
	// 				parent := rn.Parent
	// 				parent.RemoveChild(rn)
	// 			}
	// 		default:
	// 			{
	// 				copyChildren(rn, newNode)
	// 				replaceNode(rn, newNode)
	// 			}
	// 		}
	// 	}
	// }

	return renderHTML(p.Output, n)
}
