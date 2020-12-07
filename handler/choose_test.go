package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/strategy"
)

type fakeChooseStore struct {
	session data.Session
	client  data.Client
	login   string
}

func (s *fakeChooseStore) Login(r *http.Request) (string, error) {
	if s.login != "" {
		return s.login, nil
	}
	return "", errors.New("nope")
}

func (s *fakeChooseStore) CreateSession(session data.Session) error {
	s.session = session
	return nil
}

func (s *fakeChooseStore) Client(clientID, redirectURI string) (data.Client, error) {
	if clientID == s.client.ID && redirectURI == s.client.RedirectURI {
		return s.client, nil
	}
	return data.Client{}, errors.New("what")
}

func TestChoose(t *testing.T) {
	assert := assert.Wrap(t)

	store := &fakeChooseStore{
		client: data.Client{
			ID:          "http://client.example.com/",
			RedirectURI: "http://client.example.com/callback",
			Name:        "Client",
		},
	}
	tmpl := &mockTemplate{}

	s := httptest.NewServer(Choose("http://localhost", store, strategy.Strategies{&fakeStrategy{}}, tmpl))
	defer s.Close()

	form := url.Values{
		"me":           {"http://mE.example.com"},
		"client_id":    {"http://clIent.exAmple.com"},
		"redirect_uri": {"http://client.example.com/callback"},
		"state":        {"some-value"},
	}

	resp, err := http.Get(s.URL + "?" + form.Encode())
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusOK)

	data := tmpl.Data.(chooseCtx)
	assert(tmpl.Tmpl).Equal("choose.gotmpl")
	assert(data.ClientID).Equal("http://client.example.com/")
	assert(data.ClientName).Equal("Client")
	assert(data.Me).Equal("http://me.example.com/")
	assert(data.Skip).False()

	assert(store.session.ResponseType).Equal("id")
	assert(store.session.Me).Equal("http://me.example.com/")
	assert(store.session.ClientID).Equal("http://client.example.com/")
	assert(store.session.RedirectURI).Equal("http://client.example.com/callback")
	assert(store.session.State).Equal("some-value")
}

func TestChooseWithRecentLogin(t *testing.T) {
	assert := assert.Wrap(t)

	store := &fakeChooseStore{
		client: data.Client{
			ID:          "http://client.example.com/",
			RedirectURI: "http://client.example.com/callback",
			Name:        "Client",
		},
		login: "http://me.example.com/",
	}
	tmpl := &mockTemplate{}

	s := httptest.NewServer(Choose("http://localhost", store, strategy.Strategies{&fakeStrategy{}}, tmpl))
	defer s.Close()

	form := url.Values{
		"me":           {"http://mE.example.com"},
		"client_id":    {"http://clIent.exAmple.com"},
		"redirect_uri": {"http://client.example.com/callback"},
		"state":        {"some-value"},
	}

	resp, err := http.Get(s.URL + "?" + form.Encode())
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusOK)

	data := tmpl.Data.(chooseCtx)
	assert(tmpl.Tmpl).Equal("choose.gotmpl")
	assert(data.ClientID).Equal("http://client.example.com/")
	assert(data.ClientName).Equal("Client")
	assert(data.Me).Equal("http://me.example.com/")
	assert(data.Skip).True()

	assert(store.session.ResponseType).Equal("id")
	assert(store.session.Me).Equal("http://me.example.com/")
	assert(store.session.ClientID).Equal("http://client.example.com/")
	assert(store.session.RedirectURI).Equal("http://client.example.com/callback")
	assert(store.session.State).Equal("some-value")
}

func TestChooseWhenClientCannotBeRetrieved(t *testing.T) {
	assert := assert.Wrap(t)

	store := &fakeChooseStore{}
	tmpl := &mockTemplate{}

	s := httptest.NewServer(Choose("http://localhost", store, strategy.Strategies{&fakeStrategy{}}, tmpl))
	defer s.Close()

	form := url.Values{
		"me":           {"http://mE.example.com"},
		"client_id":    {"http://clIent.exAmple.com"},
		"redirect_uri": {"http://client.example.com/callback"},
		"state":        {"some-value"},
	}

	resp, err := http.Get(s.URL + "?" + form.Encode())
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusBadRequest)
}

func TestChooseWithBadMe(t *testing.T) {
	assert := assert.Wrap(t)

	store := &fakeChooseStore{
		client: data.Client{
			ID:          "http://client.example.com/",
			RedirectURI: "http://client.example.com/callback",
			Name:        "Client",
		},
	}
	tmpl := &mockTemplate{}

	s := httptest.NewServer(Choose("http://localhost", store, strategy.Strategies{&fakeStrategy{}}, tmpl))
	defer s.Close()

	form := url.Values{
		"me":           {"http://127.0.0.1/"},
		"client_id":    {"http://client.example.com/"},
		"redirect_uri": {"http://client.example.com/callback"},
		"state":        {"some-value"},
	}

	resp, err := http.Get(s.URL + "?" + form.Encode())
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusBadRequest)
}

func TestChooseWithBadClientID(t *testing.T) {
	assert := assert.Wrap(t)

	store := &fakeChooseStore{
		client: data.Client{
			ID:          "http://client.example.com/",
			RedirectURI: "http://client.example.com/callback",
			Name:        "Client",
		},
	}
	tmpl := &mockTemplate{}

	s := httptest.NewServer(Choose("http://localhost", store, strategy.Strategies{&fakeStrategy{}}, tmpl))
	defer s.Close()

	form := url.Values{
		"me":           {"http://me.example.com"},
		"client_id":    {"mailto:client.example.com/"},
		"redirect_uri": {"http://client.example.com/callback"},
		"state":        {"some-value"},
	}

	resp, err := http.Get(s.URL + "?" + form.Encode())
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusBadRequest)
}

func TestChooseWithBadParams(t *testing.T) {
	store := &fakeChooseStore{
		client: data.Client{
			ID:          "http://client.example.com/",
			RedirectURI: "http://client.example.com/callback",
			Name:        "Client",
		},
	}
	tmpl := &mockTemplate{}

	s := httptest.NewServer(Choose("http://localhost", store, strategy.Strategies{&fakeStrategy{}}, tmpl))
	defer s.Close()

	testCases := map[string]url.Values{
		"missing me": {
			"client_id":    {"http://client.example.com/"},
			"redirect_uri": {"http://client.example.com/callback"},
			"state":        {"some-value"},
		},
		"missing client_id": {
			"me":           {"http://me.example.com/"},
			"redirect_uri": {"http://client.example.com/callback"},
			"state":        {"some-value"},
		},
		"missing redirect_uri": {
			"me":        {"http://me.example.com/"},
			"client_id": {"http://client.example.com/"},
			"state":     {"some-value"},
		},
		"missing state": {
			"me":           {"http://me.example.com/"},
			"client_id":    {"http://client.example.com/"},
			"redirect_uri": {"http://client.example.com/callback"},
		},
		"bad response_type": {
			"response_type": {"nope"},
			"me":            {"http://me.example.com/"},
			"client_id":     {"http://client.example.com/"},
			"redirect_uri":  {"http://client.example.com/callback"},
			"state":         {"abcde"},
		},
	}

	for name, form := range testCases {
		t.Run(name, func(t *testing.T) {
			resp, err := http.Get(s.URL + "?" + form.Encode())
			assert.Nil(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestChooseForCode(t *testing.T) {
	assert := assert.Wrap(t)

	store := &fakeChooseStore{
		client: data.Client{
			ID:          "http://client.example.com/",
			RedirectURI: "http://client.example.com/callback",
			Name:        "Client",
		},
	}
	tmpl := &mockTemplate{}

	s := httptest.NewServer(Choose("http://localhost", store, strategy.Strategies{&fakeStrategy{}}, tmpl))
	defer s.Close()

	form := url.Values{
		"me":            {"http://me.example.com/"},
		"client_id":     {"http://client.example.com/"},
		"redirect_uri":  {"http://client.example.com/callback"},
		"state":         {"some-value"},
		"response_type": {"code"},
		"scope":         {"create update"},
	}

	resp, err := http.Get(s.URL + "?" + form.Encode())
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusOK)

	data := tmpl.Data.(chooseCtx)
	assert(tmpl.Tmpl).Equal("choose.gotmpl")
	assert(data.ClientID).Equal("http://client.example.com/")
	assert(data.ClientName).Equal("Client")
	assert(data.Me).Equal("http://me.example.com/")
	assert(data.Skip).False()

	assert(store.session.ResponseType).Equal("code")
	assert(store.session.Me).Equal("http://me.example.com/")
	assert(store.session.ClientID).Equal("http://client.example.com/")
	assert(store.session.RedirectURI).Equal("http://client.example.com/callback")
	assert(store.session.State).Equal("some-value")
	assert(store.session.Scope).Equal("create update")
}

func TestChooseForCodeWithPKCE(t *testing.T) {
	assert := assert.Wrap(t)

	store := &fakeChooseStore{
		client: data.Client{
			ID:          "http://client.example.com/",
			RedirectURI: "http://client.example.com/callback",
			Name:        "Client",
		},
	}
	tmpl := &mockTemplate{}

	s := httptest.NewServer(Choose("http://localhost", store, strategy.Strategies{&fakeStrategy{}}, tmpl))
	defer s.Close()

	form := url.Values{
		"me":                    {"http://me.example.com/"},
		"client_id":             {"http://client.example.com/"},
		"redirect_uri":          {"http://client.example.com/callback"},
		"code_challenge":        {"some-base64-string"},
		"code_challenge_method": {"S256"},
		"state":                 {"some-value"},
		"response_type":         {"code"},
		"scope":                 {"create update"},
	}

	resp, err := http.Get(s.URL + "?" + form.Encode())
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusOK)

	data := tmpl.Data.(chooseCtx)
	assert(tmpl.Tmpl).Equal("choose.gotmpl")
	assert(data.ClientID).Equal("http://client.example.com/")
	assert(data.ClientName).Equal("Client")
	assert(data.Me).Equal("http://me.example.com/")
	assert(data.Skip).False()

	assert(store.session.ResponseType).Equal("code")
	assert(store.session.Me).Equal("http://me.example.com/")
	assert(store.session.ClientID).Equal("http://client.example.com/")
	assert(store.session.RedirectURI).Equal("http://client.example.com/callback")
	assert(store.session.CodeChallenge).Equal("some-base64-string")
	assert(store.session.CodeChallengeMethod).Equal("S256")
	assert(store.session.State).Equal("some-value")
	assert(store.session.Scope).Equal("create update")
}

func TestChooseForCodeWithBadParams(t *testing.T) {
	store := &fakeChooseStore{
		client: data.Client{
			ID:          "http://client.example.com/",
			RedirectURI: "http://client.example.com/callback",
			Name:        "Client",
		},
	}
	tmpl := &mockTemplate{}

	s := httptest.NewServer(Choose("http://localhost", store, strategy.Strategies{&fakeStrategy{}}, tmpl))
	defer s.Close()

	testCases := map[string]url.Values{
		"missing me": {
			"client_id":     {"http://client.example.com/"},
			"redirect_uri":  {"http://client.example.com/callback"},
			"state":         {"some-value"},
			"response_type": {"code"},
			"scope":         {"create update"},
		},
		"missing client_id": {
			"me":            {"http://me.example.com/"},
			"redirect_uri":  {"http://client.example.com/callback"},
			"state":         {"some-value"},
			"response_type": {"code"},
			"scope":         {"create update"},
		},
		"missing redirect_uri": {
			"me":            {"http://me.example.com/"},
			"client_id":     {"http://client.example.com/"},
			"state":         {"some-value"},
			"response_type": {"code"},
			"scope":         {"create update"},
		},
		"missing state": {
			"me":            {"http://me.example.com/"},
			"client_id":     {"http://client.example.com/"},
			"redirect_uri":  {"http://client.example.com/callback"},
			"response_type": {"code"},
			"scope":         {"create update"},
		},
		"missing scope": {
			"me":            {"http://me.example.com/"},
			"client_id":     {"http://client.example.com/"},
			"redirect_uri":  {"http://client.example.com/callback"},
			"state":         {"some-value"},
			"response_type": {"code"},
		},
	}

	for name, form := range testCases {
		t.Run(name, func(t *testing.T) {
			resp, err := http.Get(s.URL + "?" + form.Encode())
			assert.Nil(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}
