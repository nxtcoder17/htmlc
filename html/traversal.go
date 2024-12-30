package html

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

func parseReader(r io.Reader, traverse func(*html.Node) error) error {
	n, err := html.Parse(r)
	if err != nil {
		return err
	}

	return traverse(n)
}

func getChildren(n *html.Node) []*html.Node {
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if strings.TrimSpace(c.Data) == "" {
			// INFO: omiting empty node
			continue
		}
		result = append(result, c)
	}

	return result
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
