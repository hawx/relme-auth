package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/strategy"
)

type fakeAuthStore struct {
	session data.Session
}

func (s *fakeAuthStore) Session(me string) (data.Session, error) {
	if me == s.session.Me {
		return s.session, nil
	}
	return data.Session{}, errors.New("nope")
}

func (s *fakeAuthStore) SetProvider(me, provider, profileURI string) error {
	return nil
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

	assert := assert.New(t)
	authStore := &fakeAuthStore{}
	strat := &fakeStrategy{}

	r := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(sURL))
	}))
	defer r.Close()
	rURL = r.URL

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(rURL))
	}))
	defer s.Close()
	sURL = s.URL

	a := httptest.NewServer(Auth(authStore, strategy.Strategies{strat}, http.DefaultClient))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	authStore.session = data.Session{
		Me:          s.URL,
		ClientID:    "https://example.com/",
		RedirectURI: "https://example.com/redirect",
		State:       "shared state",
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me":           {s.URL},
		"provider":     {strat.Name()},
		"profile":      {"https://me.example.com"},
		"client_id":    {"https://example.com/"},
		"redirect_uri": {"https://example.com/redirect"},
		"state":        {"shared state"},
	}.Encode(), nil)
	assert.Nil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(err)

	assert.Equal("https://example.com/redirect", resp.Header.Get("Location"))
}

func TestAuthWithEvilRedirect(t *testing.T) {
	var rURL, sURL string

	assert := assert.New(t)
	authStore := &fakeAuthStore{}
	strat := &fakeStrategy{}

	r := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(sURL))
	}))
	defer r.Close()
	rURL = r.URL

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(rURL))
	}))
	defer s.Close()
	sURL = s.URL

	a := httptest.NewServer(Auth(authStore, strategy.Strategies{strat}, http.DefaultClient))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	authStore.session = data.Session{
		Me:          s.URL,
		ClientID:    "https://example.com/",
		RedirectURI: "https://not.example.com/redirect",
		State:       "shared state",
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me":           {s.URL},
		"provider":     {strat.Name()},
		"profile":      {"https://me.example.com"},
		"client_id":    {"https://example.com/"},
		"redirect_uri": {"https://not.example.com/redirect"},
	}.Encode(), nil)
	assert.Nil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(err)
	assert.Equal(http.StatusBadRequest, resp.StatusCode)
}

func TestAuthWithEvilRedirectThatIsWhitelistedInHeader(t *testing.T) {
	var rURL, sURL string

	assert := assert.New(t)
	authStore := &fakeAuthStore{}
	strat := &fakeStrategy{}

	c := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://example.com/redirect>; rel="redirect_uri"`)
	}))
	defer c.Close()

	r := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(sURL))
	}))
	defer r.Close()
	rURL = r.URL

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(rURL))
	}))
	defer s.Close()
	sURL = s.URL

	a := httptest.NewServer(Auth(authStore, strategy.Strategies{strat}, http.DefaultClient))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	authStore.session = data.Session{
		Me:          s.URL,
		ClientID:    c.URL,
		RedirectURI: "https://example.com/redirect",
		State:       "shared state",
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me":           {s.URL},
		"provider":     {strat.Name()},
		"profile":      {"https://me.example.com"},
		"client_id":    {c.URL},
		"redirect_uri": {"https://example.com/redirect"},
	}.Encode(), nil)
	assert.Nil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(err)
	assert.Equal("https://example.com/redirect", resp.Header.Get("Location"))
}

func TestAuthWithEvilRedirectThatIsWhitelistedInLink(t *testing.T) {
	var rURL, sURL string

	assert := assert.New(t)
	authStore := &fakeAuthStore{}
	strat := &fakeStrategy{}

	c := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html>
<head>
<link rel="redirect_uri" href="https://example.com/redirect" />
</head>
</html>`)
	}))
	defer c.Close()

	r := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(sURL))
	}))
	defer r.Close()
	rURL = r.URL

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(rURL))
	}))
	defer s.Close()
	sURL = s.URL

	a := httptest.NewServer(Auth(authStore, strategy.Strategies{strat}, http.DefaultClient))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	authStore.session = data.Session{
		Me:          s.URL,
		ClientID:    c.URL,
		RedirectURI: "https://example.com/redirect",
		State:       "shared state",
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me":           {s.URL},
		"provider":     {strat.Name()},
		"profile":      {"https://me.example.com"},
		"client_id":    {c.URL},
		"redirect_uri": {"https://example.com/redirect"},
	}.Encode(), nil)
	assert.Nil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(err)
	assert.Equal("https://example.com/redirect", resp.Header.Get("Location"))
}

func TestAuthWhenNoMatchingStrategies(t *testing.T) {
	var rURL, sURL string

	assert := assert.New(t)
	authStore := &fakeAuthStore{}
	strat := &falseStrategy{}

	r := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(sURL))
	}))
	defer r.Close()
	rURL = r.URL

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testPage(rURL))
	}))
	defer s.Close()
	sURL = s.URL

	a := httptest.NewServer(Auth(authStore, strategy.Strategies{strat}, http.DefaultClient))
	defer a.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me": {s.URL},
	}.Encode(), nil)
	assert.Nil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(err)
	assert.Equal(http.StatusBadRequest, resp.StatusCode)
}
