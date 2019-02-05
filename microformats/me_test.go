package microformats

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hawx.me/code/assert"
)

var client = &RelMe{Client: http.DefaultClient}

func page(links ...string) string {
	s := `<!doctype html>
<html>
<head>

</head>
<body>`
	for _, link := range links {
		s += `<a rel="me" href="` + link + `">ok</a>`
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

func TestMe(t *testing.T) {
	assert := assert.New(t)

	var meSite, otherSite, missingSite *httptest.Server

	missingSite = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, page("what"))
	}))
	defer missingSite.Close()

	otherSite = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, page(meSite.URL))
	}))
	defer otherSite.Close()

	meSite = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, page(otherSite.URL, missingSite.URL, "http://localhost/unknown"))
	}))
	defer meSite.Close()

	eventsCh := Me(meSite.URL)

	event, ok, timedOut := getEvent(eventsCh)
	if assert.False(timedOut) {
		assert.True(ok)
		assert.Equal(Found, event.Type)
		assert.Equal(otherSite.URL, event.Link)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert.False(timedOut) {
		assert.True(ok)
		assert.Equal(Found, event.Type)
		assert.Equal(missingSite.URL, event.Link)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert.False(timedOut) {
		assert.True(ok)
		assert.Equal(Found, event.Type)
		assert.Equal("http://localhost/unknown", event.Link)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert.False(timedOut) {
		assert.True(ok)
		assert.Equal(Verified, event.Type)
		assert.Equal(otherSite.URL, event.Link)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert.False(timedOut) {
		assert.True(ok)
		assert.Equal(Unverified, event.Type)
		assert.Equal(missingSite.URL, event.Link)
	}

	event, ok, timedOut = getEvent(eventsCh)
	if assert.False(timedOut) {
		assert.True(ok)
		assert.Equal(Error, event.Type)
		assert.Equal("http://localhost/unknown", event.Link)
		assert.NotNil(event.Err)
	}

	_, ok, timedOut = getEvent(eventsCh)
	if assert.False(timedOut) {
		assert.False(ok)
	}
}

func testMePage(link string) string {
	return `
<!doctype html>
<html>
<head>

</head>
<body>
  <a rel="me" href="` + link + `">ok</a>
  <a rel="me" href="http://localhost/unknown">what</a>
</body>
`
}

func testAuthPage(link, badLink string) string {
	return `
<!doctype html>
<html>
<head>

</head>
<body>
  <a rel="me" href="` + badLink + `">ok</a>
  <a rel="me authn" href="` + link + `">ok</a>
  <a rel="me" href="http://localhost/unknown">what</a>
</body>
`
}

func TestFindAuth(t *testing.T) {
	assert := assert.New(t)

	var me, good, bad *httptest.Server

	good = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testMePage(me.URL))
	}))
	defer good.Close()

	bad = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testMePage(me.URL))
	}))
	defer bad.Close()

	me = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testAuthPage(good.URL, bad.URL))
	}))
	defer me.Close()

	links, err := client.FindAuth(me.URL)
	assert.Nil(err)

	if assert.Len(links, 1) {
		assert.Equal(links[0], good.URL)
	}
}

func TestFindAuthWhenNoAuthnRels(t *testing.T) {
	assert := assert.New(t)

	var me, good *httptest.Server

	good = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testMePage(me.URL))
	}))
	defer good.Close()

	me = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testMePage(good.URL))
	}))
	defer me.Close()

	links, err := client.FindAuth(me.URL)
	assert.Nil(err)

	if assert.Len(links, 2) {
		assert.Equal(links[0], good.URL)
		assert.Equal(links[1], "http://localhost/unknown")
	}
}

func TestFind(t *testing.T) {
	assert := assert.New(t)

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
		fmt.Fprintf(w, html)
	}))
	defer s.Close()

	links, err := client.Find(s.URL)
	assert.Nil(err)

	if assert.Len(links, 2) {
		assert.Equal(links[0], "https://example.com/a")
		assert.Equal(links[1], "https://example.com/b")
	}
}

func TestLinksTo(t *testing.T) {
	assert := assert.New(t)

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
		fmt.Fprintf(w, html)
	}))
	defer s.Close()

	ok, err := client.LinksTo(s.URL, "https://example.com/a")
	assert.Nil(err)
	assert.True(ok)
	ok, err = client.LinksTo(s.URL, "https://example.com/a/")
	assert.Nil(err)
	assert.True(ok)
	ok, err = client.LinksTo(s.URL, "http://example.com/a")
	assert.Nil(err)
	assert.True(ok)

	ok, err = client.LinksTo(s.URL, "https://example.com/b")
	assert.Nil(err)
	assert.False(ok)
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

	assert := assert.New(t)
	var twitterURL string

	// my homepage links to my twitter
	homepage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testMePage(twitterURL))
	}))
	defer homepage.Close()

	// tco redirects to my homepage
	tco := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, homepage.URL, http.StatusFound)
	}))
	defer tco.Close()

	// twitter has a link to tco
	twitter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testMePage(tco.URL))
	}))
	twitterURL = twitter.URL
	defer twitter.Close()

	// then we can verify that my homepage links twitter
	ok, err := client.LinksTo(homepage.URL, twitter.URL)
	assert.Nil(err)
	assert.True(ok)

	// and twitter links to my homepage
	ok, err = client.LinksTo(twitter.URL, homepage.URL)
	assert.Nil(err)
	assert.True(ok)
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

	assert := assert.New(t)
	var twitterURL string

	// my homepage links to my twitter
	homepage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testMePage(twitterURL))
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
		fmt.Fprintf(w, testMePage(tco.URL))
	}))
	twitterURL = twitter.URL
	defer twitter.Close()

	// then we can verify that my homepage links twitter
	ok, err := client.LinksTo(homepage.URL, twitter.URL)
	assert.Nil(err)
	assert.True(ok)

	// and twitter links to my homepage
	ok, err = client.LinksTo(twitter.URL, homepage.URL)
	assert.Nil(err)
	assert.True(ok)

	// and we can verify that my short homepage links twitter
	ok, err = client.LinksTo(shortHomepage.URL, twitter.URL)
	assert.Nil(err)
	assert.True(ok)

	// and twitter links to my short homepage
	ok, err = client.LinksTo(twitter.URL, shortHomepage.URL)
	assert.Nil(err)
	assert.True(ok)
}
