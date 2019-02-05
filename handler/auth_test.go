package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/data/memory"
	"hawx.me/code/relme-auth/strategy"
)

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
	authStore := memory.New()

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

	authStore.Save(&data.Session{
		Me:          s.URL,
		ClientID:    "https://example.com/",
		RedirectURI: "https://example.com/redirect",
		State:       "shared state",
	})

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
	authStore := memory.New()
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

	authStore.Save(&data.Session{
		Me:          s.URL,
		ClientID:    "https://example.com/",
		RedirectURI: "https://not.example.com/redirect",
		State:       "shared state",
	})

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
	authStore := memory.New()
	strat := &fakeStrategy{}

	c := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://example.com/redirect>; rel="redirect_uri"`)
	}))
	defer c.Close()

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

	authStore.Save(&data.Session{
		Me:          s.URL,
		ClientID:    c.URL,
		RedirectURI: "https://example.com/redirect",
		State:       "shared state",
	})

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
	authStore := memory.New()
	strat := &fakeStrategy{}

	c := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html>
<head>
<link rel="redirect_uri" href="https://example.com/redirect" />
</head>
</html>`)
	}))
	defer c.Close()

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

	authStore.Save(&data.Session{
		Me:          s.URL,
		ClientID:    c.URL,
		RedirectURI: "https://example.com/redirect",
		State:       "shared state",
	})

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
	authStore := memory.New()
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
	assert.Nil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	assert.Nil(err)
	assert.Equal(http.StatusBadRequest, resp.StatusCode)
}
