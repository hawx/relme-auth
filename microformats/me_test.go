package microformats

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hawx.me/code/assert"
)

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
