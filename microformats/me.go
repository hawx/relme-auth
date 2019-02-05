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
		profileLinks, err := client.FindAuth(profile)
		if err != nil {
			eventCh <- Event{Type: Error, Err: err}
			close(eventCh)
			return
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
func (me *RelMe) FindAuth(profile string) (links []string, err error) {
	req, err := http.NewRequest("GET", profile, nil)
	if err != nil {
		return
	}

	resp, err := me.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return parseLinks(resp.Body, isRelAuthn, isRelMe)
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

	return parseLinks(resp.Body, isRelMe)
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

func parseLinks(r io.Reader, preds ...func(*html.Node) bool) (links []string, err error) {
	root, err := html.Parse(r)
	if err != nil {
		return
	}

	for _, pred := range preds {
		rels := searchAll(root, pred)
		for _, node := range rels {
			links = append(links, getAttr(node, "href"))
		}
		if len(links) > 0 {
			return
		}
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
