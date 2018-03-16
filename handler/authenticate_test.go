package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/state"
	"hawx.me/code/relme-auth/strategy"
)

type fakeStrategy struct {
	match        *url.URL
	expectedLink string
	form         url.Values
}

func (s *fakeStrategy) Match(me *url.URL) bool {
	s.match = me
	return true
}

func (s *fakeStrategy) Redirect(expectedLink string) (redirectURL string, err error) {
	s.expectedLink = expectedLink
	return "https://redirect.example.com", nil
}

func (s *fakeStrategy) Callback(form url.Values) (string, error) {
	s.form = form
	return "me", nil
}

type falseStrategy struct{}

func (s *falseStrategy) Match(me *url.URL) bool {
	return false
}

func (s *falseStrategy) Redirect(expectedLink string) (redirectURL string, err error) {
	return "https://redirect.example.com", nil
}

func (s *falseStrategy) Callback(form url.Values) (string, error) {
	return "me", nil
}

func testPage(link string) string {
	return `
<!doctype html>
<html>
<head>

</head>
<body>
  <a rel="me" href="` + link + `">ok</a>
</body>
`
}

func TestAuthenticate(t *testing.T) {
	var rURL, sURL string

	authStore := state.NewStore()
	strat := &fakeStrategy{}

	r := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testPage(sURL))
	}))
	defer r.Close()
	rURL = r.URL

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testPage(rURL))
	}))
	defer s.Close()
	sURL = s.URL

	a := httptest.NewServer(Authenticate(authStore, strategy.Strategies{strat}))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("POST", a.URL, strings.NewReader(url.Values{
		"me": {s.URL},
	}.Encode()))
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(t, err)

	if assert.NotNil(t, strat.match) {
		assert.Equal(t, r.URL, strat.match.String())
	}

	assert.Equal(t, "https://redirect.example.com", resp.Header.Get("Location"))
}

func TestAuthenticateWhenNoMatchingStrategies(t *testing.T) {
	var rURL, sURL string

	authStore := state.NewStore()
	strat := &falseStrategy{}

	r := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testPage(sURL))
	}))
	defer r.Close()
	rURL = r.URL

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testPage(rURL))
	}))
	defer s.Close()
	sURL = s.URL

	a := httptest.NewServer(Authenticate(authStore, strategy.Strategies{strat}))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("POST", a.URL, strings.NewReader(url.Values{
		"me": {s.URL},
	}.Encode()))
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, "/no-strategies", resp.Header.Get("Location"))
}
