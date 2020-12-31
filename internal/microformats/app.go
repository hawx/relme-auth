package microformats

import (
	"errors"
	"io"
	"net/url"

	"willnorris.com/go/microformats"
)

// ErrNoApp is used to signal when no app microformat exists.
var ErrNoApp = errors.New("no h-x-app or h-app could be found")

type App struct {
	Name         string
	URL          string
	RedirectURIs []string
}

func ParseApp(r io.Reader, baseURL *url.URL) (app App, err error) {
	var appExists bool
	data := microformats.Parse(r, baseURL)

	app.RedirectURIs = data.Rels["redirect_uri"]

	for _, item := range data.Items {
		if hasEitherType(item.Type, "h-app", "h-x-app") {
			appExists = true

			if len(item.Properties["name"]) == 1 {
				name, ok := item.Properties["name"][0].(string)
				if ok {
					app.Name = name
				}
			}

			if len(item.Properties["url"]) == 1 {
				url, ok := item.Properties["url"][0].(string)
				if ok {
					app.URL = url
				}
			}
		}
	}

	if app.Name == "" {
		app.Name = app.URL
	}

	if !appExists {
		err = ErrNoApp
	}

	return
}

func hasEitherType(list []string, a, b string) bool {
	for _, item := range list {
		if item == a || item == b {
			return true
		}
	}
	return false
}
