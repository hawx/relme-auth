package microformats

import (
	"errors"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// ErrNoApp is used to signal when no app microformat exists.
var ErrNoApp = errors.New("no h-x-app or h-app could be found")

// HApp attempts to find the name and url provided by the h-app or h-x-app
// microformat.
func HApp(r io.Reader) (name string, url string, err error) {
	root, err := html.Parse(r)
	if err != nil {
		return
	}

	hApp := searchAll(root, hasEitherClass("h-x-app", "h-app"))
	if len(hApp) == 0 {
		err = ErrNoApp
		return
	}

	uURL := searchAll(hApp[0], hasClass("u-url"))
	if len(uURL) != 0 {
		url = getAttr(uURL[0], "href")
	}

	pName := searchAll(hApp[0], hasClass("p-name"))
	if len(pName) != 0 {
		name = textOf(pName[0])
	} else if url != "" {
		name = url
	}

	return
}

// RedirectURIs finds whitelisted redirect_uris from the Reader.
func RedirectURIs(r io.Reader) []string {
	var whitelist []string

	root, err := html.Parse(r)
	if err != nil {
		return whitelist
	}

	redirectLinks := searchAll(root, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "link" {
			rels := strings.Fields(getAttr(node, "rel"))
			for _, rel := range rels {
				if rel == "redirect_uri" {
					return true
				}
			}
		}

		return false
	})

	for _, node := range redirectLinks {
		whitelist = append(whitelist, getAttr(node, "href"))
	}

	return whitelist
}
