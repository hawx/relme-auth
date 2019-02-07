package microformats

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"hawx.me/code/relme-auth/strategy"
)

type matching interface {
	IsAllowed(link string) (found strategy.Strategy, ok bool)
}

// EventType defines what has occured.
type EventType uint

const (
	// Error means something went wrong when trying to request a URL.
	Error EventType = iota
	// Found means a new link has been found that could be used for
	// authentication.
	Found
	// Verified means a link has been confirmed as usable for authentication.
	Verified
	// Unverified means a previously found link can't be used for authentication.
	Unverified
	// PGP means a pgpkey has been found that can be used for authentication.
	PGP
)

// Event is emitted by Me as new links are found and verified.
type Event struct {
	Type EventType
	Link string
	Err  error
}

// Me requests profile, then finds all links that can be used to authenticate
// the user.
func (client *RelMe) Me(profile string, strategies matching) <-chan Event {
	eventCh := make(chan Event)

	go func() {
		profileLinks, pgpkey, err := client.FindAuth(profile)
		if err != nil {
			eventCh <- Event{Type: Error, Err: err}
			close(eventCh)
			return
		}

		if pgpkey != "" {
			eventCh <- Event{Type: PGP, Link: pgpkey}
		}

		var allowedLinks []string
		for _, link := range profileLinks {
			if _, ok := strategies.IsAllowed(link); ok {
				eventCh <- Event{Type: Found, Link: link}
				allowedLinks = append(allowedLinks, link)
			}
		}

		for _, link := range allowedLinks {
			ok, err := client.LinksTo(link, profile)

			if err != nil {
				eventCh <- Event{Type: Error, Link: link, Err: err}
			} else if ok {
				eventCh <- Event{Type: Verified, Link: link}
			} else {
				eventCh <- Event{Type: Unverified, Link: link}
			}
		}

		close(eventCh)
	}()

	return eventCh
}

type RelMe struct {
	Client           *http.Client
	NoRedirectClient *http.Client
}

// FindAuth takes a profile URL and returns a list of all hrefs in <a rel="me
// authn"/> elements on the page that also link back to the profile, if none
// exist it fallsback to using hrefs in <a rel="me"/> elements as FindVerified
// does.
func (me *RelMe) FindAuth(profile string) (links []string, pgpkey string, err error) {
	req, err := http.NewRequest("GET", profile, nil)
	if err != nil {
		return
	}

	resp, err := me.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return parseProfileLinks(profile, resp.Body)
}

// Find takes a profile URL and returns a list of all hrefs in <a rel="me"/>
// elements on the page.
func (me *RelMe) Find(profile string) (links []string, err error) {
	req, err := http.NewRequest("GET", profile, nil)
	if err != nil {
		return
	}

	resp, err := me.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return parseLinks(resp.Body)
}

// LinksTo takes a remote profile URL and checks whether any of the hrefs in <a
// rel="me"/> elements match the test URL.
func (me *RelMe) LinksTo(remote, test string) (ok bool, err error) {
	testURL, err := url.Parse(test)
	if err != nil {
		return
	}

	testRedirects, err := me.follow(testURL)
	if err != nil {
		return
	}

	links, err := me.Find(remote)
	if err != nil {
		return
	}

	for _, link := range links {
		linkURL, err := url.Parse(link)
		if err != nil {
			continue
		}

		linkRedirects, err := me.follow(linkURL)
		if err != nil {
			continue
		}

		if compare(linkRedirects, testRedirects) {
			return true, nil
		}
	}

	return false, nil
}

func normalizeInPlace(urls []string) {
	for i, a := range urls {
		aURL, err := url.Parse(a)
		if err != nil {
			continue
		}
		aURL.Scheme = "https"

		urls[i] = strings.TrimRight(aURL.String(), "/")
	}
}

func compare(as, bs []string) bool {
	normalizeInPlace(as)
	normalizeInPlace(bs)

	for _, a := range as {
		for _, b := range bs {
			if a == b {
				return true
			}
		}
	}

	return false
}

func (me *RelMe) follow(remote *url.URL) (redirects []string, err error) {
	previous := map[string]struct{}{}
	current := remote

	for {
		redirects = append(redirects, current.String())

		req, err := http.NewRequest("GET", current.String(), nil)
		if err != nil {
			break
		}
		previous[current.String()] = struct{}{}

		resp, err := me.NoRedirectClient.Do(req)
		if err != nil {
			break
		}
		defer resp.Body.Close()

		if resp.StatusCode < 300 || resp.StatusCode >= 400 {
			break
		}

		current, err = current.Parse(resp.Header.Get("Location"))
		if err != nil {
			break
		}

		if _, ok := previous[current.String()]; ok {
			break
		}
	}

	return
}

func parseProfileLinks(profile string, r io.Reader) (links []string, pgpkey string, err error) {
	root, err := html.Parse(r)
	if err != nil {
		return
	}

	pgpkey, keyWasAuthn := findPGPKey(profile, root)
	if keyWasAuthn {
		// only find authn me links
		rels := searchAll(root, isRelAuthn)
		for _, node := range rels {
			links = append(links, getAttr(node, "href"))
		}
		return
	}

	rels := searchAll(root, isRelAuthn)
	for _, node := range rels {
		links = append(links, getAttr(node, "href"))
	}
	// don't return the key as it wasn't authn
	if len(links) > 0 {
		return links, "", nil
	}

	rels = searchAll(root, isRelMe)
	for _, node := range rels {
		links = append(links, getAttr(node, "href"))
	}
	return
}

func parseLinks(r io.Reader) (links []string, err error) {
	root, err := html.Parse(r)
	if err != nil {
		return
	}

	rels := searchAll(root, isRelMe)
	for _, node := range rels {
		links = append(links, getAttr(node, "href"))
	}
	if len(links) > 0 {
		return
	}

	return
}

func findPGPKey(profile string, root *html.Node) (key string, wasAuthn bool) {
	searchAll(root, func(node *html.Node) bool {
		if node.Type == html.ElementNode && (node.Data == "a" || node.Data == "link") {
			var hasKey, hasAuthn bool

			rels := strings.Fields(getAttr(node, "rel"))
			for _, rel := range rels {
				if rel == "pgpkey" {
					hasKey = true
				}
				if rel == "authn" {
					hasAuthn = true
				}
				if hasKey && hasAuthn {
					key = getAttr(node, "href")
					wasAuthn = true
					return true
				}
			}

			if hasKey {
				key = getAttr(node, "href")
				wasAuthn = hasAuthn
				return true
			}
		}

		return false
	})

	if key != "" {
		profileURL, err := url.Parse(profile)
		if err != nil {
			return "", false
		}
		abspgpkey, err := profileURL.Parse(key)
		if err != nil {
			return "", false
		}
		key = abspgpkey.String()
	}

	return
}

func isRelMe(node *html.Node) bool {
	if node.Type == html.ElementNode && (node.Data == "a" || node.Data == "link") {
		rels := strings.Fields(getAttr(node, "rel"))
		for _, rel := range rels {
			if rel == "me" {
				return true
			}
		}
	}

	return false
}

func isRelAuthn(node *html.Node) bool {
	var me, authn bool

	if node.Type == html.ElementNode && (node.Data == "a" || node.Data == "link") {
		rels := strings.Fields(getAttr(node, "rel"))
		for _, rel := range rels {
			if rel == "me" {
				me = true
			}
			if rel == "authn" {
				authn = true
			}
			if me && authn {
				return true
			}
		}
	}

	return false
}
