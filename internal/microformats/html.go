package microformats

import (
	"strings"

	"golang.org/x/net/html"
)

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
	return hasAttr("class", func(attrs []string) bool {
		for _, a := range attrs {
			if a == name {
				return true
			}
		}

		return false
	})
}

func hasEitherClass(classA, classB string) func(*html.Node) bool {
	return hasAttr("class", func(attrs []string) bool {
		for _, a := range attrs {
			if a == classA || a == classB {
				return true
			}
		}

		return false
	})
}

func hasAttr(attr string, pred func([]string) bool) func(*html.Node) bool {
	return func(node *html.Node) bool {
		if node.Type == html.ElementNode {
			attrs := strings.Split(getAttr(node, attr), " ")
			return pred(attrs)
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
