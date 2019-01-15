package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data/memory"
	"hawx.me/code/relme-auth/strategy"
)

type fakeStrategy struct {
	match        *url.URL
	expectedLink string
	form         url.Values
}

func (fakeStrategy) Name() string {
	return "fake"
}

func (s *fakeStrategy) Match(me *url.URL) bool {
	s.match = me
	return true
}

func (s *fakeStrategy) Redirect(expectedLink string) (redirectURL string, err error) {
	s.expectedLink = expectedLink
	return "https://example.com/redirect", nil
}

func (s *fakeStrategy) Callback(form url.Values) (string, error) {
	s.form = form
	return "me", nil
}

type falseStrategy struct{}

func (falseStrategy) Name() string {
	return "false"
}

func (falseStrategy) Match(me *url.URL) bool {
	return false
}

func (falseStrategy) Redirect(expectedLink string) (redirectURL string, err error) {
	return "https://example.com/redirect", nil
}

func (falseStrategy) Callback(form url.Values) (string, error) {
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

func TestAuth(t *testing.T) {
	var rURL, sURL string

	authStore := memory.NewStore()
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

	a := httptest.NewServer(Auth(authStore, strategy.Strategies{strat}))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me":           {s.URL},
		"provider":     {strat.Name()},
		"profile":      {"https://me.example.com"},
		"client_id":    {"https://example.com/"},
		"redirect_uri": {"https://example.com/redirect"},
	}.Encode(), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(t, err)

	assert.Equal(t, "https://example.com/redirect", resp.Header.Get("Location"))
}

func TestAuthWithEvilRedirect(t *testing.T) {
	var rURL, sURL string

	authStore := memory.NewStore()
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

	a := httptest.NewServer(Auth(authStore, strategy.Strategies{strat}))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me":           {s.URL},
		"provider":     {strat.Name()},
		"profile":      {"https://me.example.com"},
		"client_id":    {"https://example.com/"},
		"redirect_uri": {"https://notexample.com/redirect"},
	}.Encode(), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAuthWhenNoMatchingStrategies(t *testing.T) {
	var rURL, sURL string

	authStore := memory.NewStore()
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

	a := httptest.NewServer(Auth(authStore, strategy.Strategies{strat}))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me": {s.URL},
	}.Encode(), nil)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
