package microformats

import (
	"io"

	"golang.org/x/net/html"
)

func HApp(r io.Reader) (name string, uri string, err error) {
	root, err := html.Parse(r)
	if err != nil {
		return
	}

	hApp := searchAll(root, hasEitherClass("h-x-app", "h-app"))
	if len(hApp) != 1 {
		return
	}

	pName := searchAll(hApp[0], hasClass("p-name"))
	if len(pName) != 1 {
		return
	}

	return textOf(pName[0]), "", nil
}
