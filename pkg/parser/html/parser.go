package html

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	textTemplate "text/template"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var logger *slog.Logger

var verboseDebugging bool

func init() {
	// l := log.New(log.ShowCallerInfo())
	logger = slog.Default()
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

func findChildrenPlaceholderNode(n *html.Node) *html.Node {
	if n.Type == html.ElementNode {
		logger.Debug("find HEAD element", "type", n.Type, "data", n.Data, "attr", n.Attr, "data-atom", n.DataAtom)
		switch n.Data {
		case "children":
			return n
		}
	}
	// Recursively process child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if n := findChildrenPlaceholderNode(c); n != nil {
			return n
		}
	}

	return nil
}

func parseHTML(n *html.Node, onTargetNodeFound func(node *html.Node)) error {
	if n.DataAtom == atom.Svg {
		return nil
	}

	if n.Type == html.ElementNode && !HTMLTags.Has(n.Data) {
		if strings.ToLower(n.Data) != "children" {
			// logNode("target-node", n)
			onTargetNodeFound(n)
			return nil
		}
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
		"children": func() string {
			return ""
		},
		// "__param__": func(k, v string) string {
		// 	return "/* comment */"
		// },
	})
	if _, err := t.Parse(string(b)); err != nil {
		return nil, err
	}

	if len(t.Templates()) > 1 {
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
	return nil, fmt.Errorf("failed to parse html node :)")
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
	if verboseDebugging {
		logNode(":) I HAVE BEEN CALLED", n)
	}

	var replaceNodes []*html.Node
	onTargetNodeFound := func(n *html.Node) {
		replaceNodes = append(replaceNodes, n)
	}

	if err := parseHTML(n, onTargetNodeFound); err != nil {
		return nil, err
	}

	headEl := findHeadElement(n)

	for _, rn := range replaceNodes {
		if verboseDebugging {
			logNode(":) I HAVE to replace", rn)
		}
		component, err := getComponent(rn.Data, htmlAttrsToMap(rn.Attr))
		if err != nil {
			return nil, err
		}

		b := new(bytes.Buffer)

		if err := component.Render(b); err != nil {
			return nil, err
		}

		// logger.Info("debugging", "rendered component",  b.String())

		newNode, err := parseWithFragments(b)
		if err != nil {
			return nil, err
		}

		newNode, err = parseHTMLAndTranspile(newNode, t, getComponent)
		if err != nil {
			return nil, err
		}

		if verboseDebugging {
			fmt.Println("------------------------------")
			renderHTML(os.Stdout, newNode)
			fmt.Println("------------------------------")
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
				// INFO: finds the <Children /> node, and replaces it with the real component children
				if childrenNode := findChildrenPlaceholderNode(newNode); childrenNode != nil {
					childparent := childrenNode.Parent
					copyChildren(rn, childparent, childrenNode)
					childparent.RemoveChild(childrenNode)

					newNode, err = parseHTMLAndTranspile(newNode, t, getComponent)
					if err != nil {
						return nil, err
					}
				} else {
					copyChildren(rn, newNode)

					newNode, err = parseHTMLAndTranspile(newNode, t, getComponent)
					if err != nil {
						return nil, err
					}
				}
				replaceNode(rn, newNode)
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

	return renderHTML(p.Output, n2)
}
