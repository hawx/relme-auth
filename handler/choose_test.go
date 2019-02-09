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
	assert := assert.New(t)

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
		"me":           {"http://me.example.com/"},
		"client_id":    {"http://client.example.com/"},
		"redirect_uri": {"http://client.example.com/callback"},
		"state":        {"some-value"},
	}

	resp, err := http.Get(s.URL + "?" + form.Encode())
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	data := tmpl.Data.(chooseCtx)
	assert.Equal("choose.gotmpl", tmpl.Tmpl)
	assert.Equal("http://client.example.com/", data.ClientID)
	assert.Equal("Client", data.ClientName)
	assert.Equal("http://me.example.com/", data.Me)

	assert.Equal("id", store.session.ResponseType)
	assert.Equal("http://me.example.com/", store.session.Me)
	assert.Equal("http://client.example.com/", store.session.ClientID)
	assert.Equal("http://client.example.com/callback", store.session.RedirectURI)
	assert.Equal("some-value", store.session.State)
}

func TestChooseForCode(t *testing.T) {
	assert := assert.New(t)

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
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	data := tmpl.Data.(chooseCtx)
	assert.Equal("choose.gotmpl", tmpl.Tmpl)
	assert.Equal("http://client.example.com/", data.ClientID)
	assert.Equal("Client", data.ClientName)
	assert.Equal("http://me.example.com/", data.Me)

	assert.Equal("code", store.session.ResponseType)
	assert.Equal("http://me.example.com/", store.session.Me)
	assert.Equal("http://client.example.com/", store.session.ClientID)
	assert.Equal("http://client.example.com/callback", store.session.RedirectURI)
	assert.Equal("some-value", store.session.State)
	assert.Equal("create update", store.session.Scope)
}
