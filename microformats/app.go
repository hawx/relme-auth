package microformats

import (
	"errors"
	"io"

	"golang.org/x/net/html"
)

// NoAppErr is used to signal when no app microformat exists.
var NoAppErr = errors.New("no h-x-app or h-app could be found")

// HApp attempts to find the name and url provided by the h-app or h-x-app
// microformat.
func HApp(r io.Reader) (name string, url string, err error) {
	root, err := html.Parse(r)
	if err != nil {
		return
	}

	hApp := searchAll(root, hasEitherClass("h-x-app", "h-app"))
	if len(hApp) == 0 {
		err = NoAppErr
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
