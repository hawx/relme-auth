package microformats

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

func HApp(r io.Reader) (name string, uri string, err error) {
	root, err := html.Parse(r)
	if err != nil {
		return
	}

	hApp := searchAll(root, hasClass("h-x-app"))
	if len(hApp) != 1 {
		return
	}

	pName := searchAll(hApp[0], hasClass("p-name"))
	if len(pName) != 1 {
		return
	}

	return textOf(pName[0]), "", nil
}

func textOf(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}

	var result string
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		result += textOf(child) + " "
	}

	return strings.TrimSpace(result)
}

func hasClass(name string) func(*html.Node) bool {
	return func(node *html.Node) bool {
		if node.Type == html.ElementNode {
			classes := strings.Split(getAttr(node, "class"), " ")
			for _, class := range classes {
				if class == name {
					return true
				}
			}
		}

		return false
	}
}

func searchAll(node *html.Node, pred func(*html.Node) bool) (results []*html.Node) {
	if pred(node) {
		results = append(results, node)
		return
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		result := searchAll(child, pred)
		if len(result) > 0 {
			results = append(results, result...)
		}
	}

	return
}

func getAttr(node *html.Node, attrName string) string {
	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}

	return ""
}
