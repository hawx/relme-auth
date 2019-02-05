package microformats

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type EventType uint

const (
	Error EventType = iota
	Found
	Verified
	Unverified
	PGP
)

type Event struct {
	Type EventType
	Link string
	Err  error
}

func Me(profile string) <-chan Event {
	eventCh := make(chan Event)
	client := RelMe{Client: http.DefaultClient}

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
		for _, link := range profileLinks {
			eventCh <- Event{Type: Found, Link: link}
		}

		for _, link := range profileLinks {
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
	Client *http.Client
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

	testRedirects, err := follow(testURL)
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

		linkRedirects, err := follow(linkURL)
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

func follow(remote *url.URL) (redirects []string, err error) {
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	previous := map[string]struct{}{}
	current := remote

	for {
		redirects = append(redirects, current.String())

		req, err := http.NewRequest("GET", current.String(), nil)
		if err != nil {
			break
		}
		previous[current.String()] = struct{}{}

		resp, err := noRedirectClient.Do(req)
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
		if node.Type == html.ElementNode && node.Data == "a" {
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
	if node.Type == html.ElementNode && node.Data == "a" {
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

	if node.Type == html.ElementNode && node.Data == "a" {
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
