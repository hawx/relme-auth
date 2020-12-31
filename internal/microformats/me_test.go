package microformats

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/internal/strategy"
)

var client = &RelMe{
	Client: http.DefaultClient,
	NoRedirectClient: &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	},
}

type A struct {
	Rel  string
	Href string
}

func testAPage(as []A, links []A) string {
	s := `<!doctype html><html><head>`
	for _, a := range links {
		s += `<link rel="` + a.Rel + `" href="` + a.Href + `" />`
	}
	s += `</head><body>`
	for _, a := range as {
		s += `<a rel="` + a.Rel + `" href="` + a.Href + `">ok</a>`
	}
	return s + `</body>`
}

func getEvent(ch <-chan Event) (Event, bool, bool) {
	select {
	case event, ok := <-ch:
		return event, ok, false
	case <-time.After(100 * time.Millisecond):
		return Event{}, false, true
	}
}

type matchingStrategy []string

func (s matchingStrategy) IsAllowed(link string) (found strategy.Strategy, ok bool) {
	for _, u := range s {
		if link == u {
			return nil, true
		}
	}
	return nil, false
}

func TestMe(t *testing.T) {
	assert := assert.Wrap(t)

	var meSite, someSite, otherSite, missingSite *httptest.Server

	missingSite = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: "what"}}, []A{}))
	}))
	defer missingSite.Close()

	someSite = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: meSite.URL}}, []A{}))
	}))
	defer someSite.Close()

	otherSite = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: meSite.URL}}, []A{}))
	}))
	defer otherSite.Close()

	meSite = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{
			{Rel: "me", Href: otherSite.URL},
			{Rel: "me", Href: missingSite.URL},
			{Rel: "me", Href: "http://localhost/unknown"},
			{Rel: "pgpkey", Href: "my-key"},
		}, []A{
			{Rel: "me", Href: someSite.URL},
			{Rel: "me", Href: "what://localhost/link"},
		}))
	}))
	defer meSite.Close()

	strategies := matchingStrategy([]string{otherSite.URL, missingSite.URL, someSite.URL, "what://localhost/link"})

	eventsCh := client.Me(meSite.URL, strategies)

	event, ok, timedOut := getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).True()
		assert(event.Type).Equal(PGP)
		assert(event.Link).Equal(meSite.URL + "/my-key")
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).True()
		assert(event.Type).Equal(Found)
		assert(event.Link).Equal(someSite.URL)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).True()
		assert(event.Type).Equal(Found)
		assert(event.Link).Equal("what://localhost/link")
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).True()
		assert(event.Type).Equal(Found)
		assert(event.Link).Equal(otherSite.URL)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).True()
		assert(event.Type).Equal(Found)
		assert(event.Link).Equal(missingSite.URL)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).True()
		assert(event.Type).Equal(Verified)
		assert(event.Link).Equal(someSite.URL)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).True()
		assert(event.Type).Equal(Error)
		assert(event.Link).Equal("what://localhost/link")
		assert(event.Err).NotNil()
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).True()
		assert(event.Type).Equal(Verified)
		assert(event.Link).Equal(otherSite.URL)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).True()
		assert(event.Type).Equal(Unverified)
		assert(event.Link).Equal(missingSite.URL)
	}

	_, ok, timedOut = getEvent(eventsCh)
	if assert(timedOut).False() {
		assert(ok).False()
	}
}

func TestFindAuth(t *testing.T) {
	assert := assert.Wrap(t)

	var me, good, bad *httptest.Server

	good = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: me.URL}}, []A{}))
	}))
	defer good.Close()

	bad = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: me.URL}}, []A{}))
	}))
	defer bad.Close()

	me = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{
			{Rel: "me authn", Href: good.URL},
			{Rel: "me", Href: bad.URL},
		}, []A{}))
	}))
	defer me.Close()

	links, pgpkey, err := client.FindAuth(me.URL)
	assert(err).Must.Nil()
	assert(pgpkey).Equal("")

	if assert(links).Len(1) {
		assert(links[0]).Equal(good.URL)
	}
}

func TestFindAuthWithPGPKey(t *testing.T) {
	assert := assert.Wrap(t)

	var me, good *httptest.Server

	good = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{}, []A{{Rel: "me", Href: me.URL}}))
	}))
	defer good.Close()

	me = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage(
			[]A{{Rel: "me", Href: good.URL}},
			[]A{{Rel: "pgpkey", Href: "/key"}},
		))
	}))
	defer me.Close()

	links, pgpkey, err := client.FindAuth(me.URL)
	assert(err).Must.Nil()
	assert(pgpkey).Equal(me.URL + "/key")

	if assert(links).Len(1) {
		assert(links[0]).Equal(good.URL)
	}
}

func TestFindAuthWithAuthnPGPKey(t *testing.T) {
	assert := assert.Wrap(t)

	var me, good, bad *httptest.Server

	good = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: me.URL}}, []A{}))
	}))
	defer good.Close()

	bad = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: me.URL}}, []A{}))
	}))
	defer bad.Close()

	me = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{
			{Rel: "authn me", Href: good.URL},
			{Rel: "me", Href: bad.URL},
			{Rel: "authn pgpkey", Href: "http://example.com/key"},
		}, []A{}))
	}))
	defer me.Close()

	links, pgpkey, err := client.FindAuth(me.URL)
	assert(err).Must.Nil()
	assert(pgpkey).Equal("http://example.com/key")

	if assert(links).Len(1) {
		assert(links[0]).Equal(good.URL)
	}
}

func TestFindAuthWithNoneAuthnPGPKey(t *testing.T) {
	assert := assert.Wrap(t)

	var me, good, bad *httptest.Server

	good = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: me.URL}}, []A{}))
	}))
	defer good.Close()

	bad = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: me.URL}}, []A{}))
	}))
	defer bad.Close()

	me = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{
			{Rel: "authn me", Href: good.URL},
			{Rel: "me", Href: bad.URL},
			{Rel: "pgpkey", Href: "http://example.com/key"},
		}, []A{}))
	}))
	defer me.Close()

	links, pgpkey, err := client.FindAuth(me.URL)
	assert(err).Must.Nil()
	assert(pgpkey).Equal("")

	if assert(links).Len(1) {
		assert(links[0]).Equal(good.URL)
	}
}

func TestFindAuthWhenNoAuthnRels(t *testing.T) {
	assert := assert.Wrap(t)

	var me, good *httptest.Server

	good = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: me.URL}}, []A{}))
	}))
	defer good.Close()

	me = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{
			{Rel: "me", Href: good.URL},
			{Rel: "me", Href: "http://localhost/unknown"},
		}, []A{}))
	}))
	defer me.Close()

	links, _, err := client.FindAuth(me.URL)
	assert(err).Must.Nil()

	if assert(links).Len(2) {
		assert(links[0]).Equal(good.URL)
		assert(links[1]).Equal("http://localhost/unknown")
	}
}

func TestFind(t *testing.T) {
	assert := assert.Wrap(t)

	html := `
<!doctype html>
<html>
<head>

</head>
<body>
  <a rel="me" href="https://example.com/a">what</a>
  <div>
    <a rel="what me ok" href="https://example.com/b">another</a>
  </div>
</body>
`

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, html)
	}))
	defer s.Close()

	links, err := client.Find(s.URL)
	assert(err).Must.Nil()

	if assert(links).Len(2) {
		assert(links[0]).Equal("https://example.com/a")
		assert(links[1]).Equal("https://example.com/b")
	}
}

func TestLinksTo(t *testing.T) {
	assert := assert.Wrap(t)

	html := `
<!doctype html>
<html>
<head>

</head>
<body>
  <a rel="me" href="https://example.com/a">ok</a>
</body>
`

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, html)
	}))
	defer s.Close()

	ok, err := client.LinksTo(s.URL, "https://example.com/a")
	assert(err).Must.Nil()
	assert(ok).True()
	ok, err = client.LinksTo(s.URL, "https://example.com/a/")
	assert(err).Must.Nil()
	assert(ok).True()
	ok, err = client.LinksTo(s.URL, "http://example.com/a")
	assert(err).Must.Nil()
	assert(ok).True()

	ok, err = client.LinksTo(s.URL, "https://example.com/b")
	assert(err).Must.Nil()
	assert(ok).False()
}

func TestLinksToWithRedirects(t *testing.T) {
	// Although this isn't stated anywhere it seems that some sites (like Twitter)
	// wrap your rel="me" link with a short version, so this needs expanding
	//
	// In the real-world example I have on my homepage https://hawx.me
	//
	//     <a rel="me" href="https://twitter.com/hawx">
	//
	// But on https://twitter.com/hawx there is only
	//
	//     <a rel="me" href="https://t.co/qsNrcG2afz">
	//
	// So I need to follow this short link to check that _any_ page it redirects
	// to matches what I expect for my homepage.

	assert := assert.Wrap(t)
	var twitterURL string

	// my homepage links to my twitter
	homepage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: twitterURL}}, []A{}))
	}))
	defer homepage.Close()

	// tco redirects to my homepage
	tco := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, homepage.URL, http.StatusFound)
	}))
	defer tco.Close()

	// twitter has a link to tco
	twitter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: tco.URL}}, []A{}))
	}))
	twitterURL = twitter.URL
	defer twitter.Close()

	// then we can verify that my homepage links twitter
	ok, err := client.LinksTo(homepage.URL, twitter.URL)
	assert(err).Must.Nil()
	assert(ok).True()

	// and twitter links to my homepage
	ok, err = client.LinksTo(twitter.URL, homepage.URL)
	assert(err).Must.Nil()
	assert(ok).True()
}

func TestLinksToWithMoreRedirects(t *testing.T) {
	// Now take the example in TestLinksToWithRedirects but pretend that both
	// links resolve to redirects. So
	//
	//     https://example.com/my-homepage
	//       302 -> https://a-really-long-domain-name.com/me
	//        me -> https://twitter.com/me
	//
	//     https://twitter.com/me
	//        me -> https://tco.com/RANDOM
	//       302 -> https://example.com/my-homepage

	assert := assert.Wrap(t)
	var twitterURL string

	// my homepage links to my twitter
	homepage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: twitterURL}}, []A{}))
	}))
	defer homepage.Close()

	// my short homepage redirects to my homepage
	shortHomepage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, homepage.URL, http.StatusFound)
	}))
	defer shortHomepage.Close()

	// tco redirects to my short homepage
	tco := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, shortHomepage.URL, http.StatusFound)
	}))
	defer tco.Close()

	// twitter has a link to tco
	twitter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testAPage([]A{{Rel: "me", Href: tco.URL}}, []A{}))
	}))
	twitterURL = twitter.URL
	defer twitter.Close()

	// then we can verify that my homepage links twitter
	ok, err := client.LinksTo(homepage.URL, twitter.URL)
	assert(err).Must.Nil()
	assert(ok).True()

	// and twitter links to my homepage
	ok, err = client.LinksTo(twitter.URL, homepage.URL)
	assert(err).Must.Nil()
	assert(ok).True()

	// and we can verify that my short homepage links twitter
	ok, err = client.LinksTo(shortHomepage.URL, twitter.URL)
	assert(err).Must.Nil()
	assert(ok).True()

	// and twitter links to my short homepage
	ok, err = client.LinksTo(twitter.URL, shortHomepage.URL)
	assert(err).Must.Nil()
	assert(ok).True()
}
