package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/internal/data"
	"hawx.me/code/relme-auth/internal/strategy"
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

	assert := assert.Wrap(t)
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
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me":           {s.URL},
		"provider":     {strat.Name()},
		"profile":      {"https://me.example.com"},
		"client_id":    {"https://example.com/"},
		"redirect_uri": {"https://example.com/redirect"},
		"state":        {"shared state"},
	}.Encode(), nil)
	assert(err).Must.Nil()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert(err).Must.Nil()
	assert(resp.Header.Get("Location")).Equal("https://example.com/redirect")
}

func TestAuthWhenSessionExpired(t *testing.T) {
	var rURL, sURL string

	assert := assert.Wrap(t)
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
		CreatedAt:   time.Now().Add(-10 * time.Minute),
		ExpiresAt:   time.Now().Add(-10 * time.Minute),
	}

	req, err := http.NewRequest("GET", a.URL+"?"+url.Values{
		"me":           {s.URL},
		"provider":     {strat.Name()},
		"profile":      {"https://me.example.com"},
		"client_id":    {"https://example.com/"},
		"redirect_uri": {"https://example.com/redirect"},
		"state":        {"shared state"},
	}.Encode(), nil)
	assert(err).Must.Nil()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusBadRequest)
	assert(resp.Header.Get("Location")).Equal("")
}

func TestAuthWhenNoMatchingStrategies(t *testing.T) {
	var rURL, sURL string

	assert := assert.Wrap(t)
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
	assert(err).Must.Nil()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusBadRequest)
}
