package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
)

func codeGenerator() string { return "my-code" }

type fakeCallbackStore struct {
	session data.Session
	code    data.Code
}

func (s *fakeCallbackStore) Session(me string) (data.Session, error) {
	if me == s.session.Me {
		return s.session, nil
	}

	return data.Session{}, errors.New("what")
}

func (s *fakeCallbackStore) CreateCode(code data.Code) error {
	s.code = code
	return nil
}

func TestCallback(t *testing.T) {
	store := &fakeCallbackStore{
		session: data.Session{
			Me:          "me",
			State:       "my-state",
			RedirectURI: "http://example.com/callback",
		},
	}

	s := httptest.NewServer(Callback(store, &fakeStrategy{}, codeGenerator))
	defer s.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	form := url.Values{
		"yes": {"ok"},
	}
	resp, err := client.Get(s.URL + "?" + form.Encode())

	assert.Nil(t, err)
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, "http://example.com/callback?code=my-code&state=my-state", resp.Header.Get("Location"))
}

func TestCallbackWhenSessionDoesNotExist(t *testing.T) {
	s := httptest.NewServer(Callback(&fakeCallbackStore{}, &fakeStrategy{}, codeGenerator))
	defer s.Close()

	form := url.Values{
		"yes": {"ok"},
	}
	resp, err := http.Get(s.URL + "?" + form.Encode())

	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCallbackWhenProviderSaysTheyAreUnauthorized(t *testing.T) {
	store := &fakeCallbackStore{
		session: data.Session{
			Me:          "me",
			State:       "my-state",
			RedirectURI: "http://example.com/callback",
		},
	}

	s := httptest.NewServer(Callback(store, &unauthorizedStrategy{}, codeGenerator))
	defer s.Close()

	form := url.Values{
		"yes": {"ok"},
	}
	resp, err := http.Get(s.URL + "?" + form.Encode())

	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCallbackWhenProviderErrors(t *testing.T) {
	store := &fakeCallbackStore{
		session: data.Session{
			Me:          "me",
			State:       "my-state",
			RedirectURI: "http://example.com/callback",
		},
	}

	s := httptest.NewServer(Callback(store, &errorStrategy{}, codeGenerator))
	defer s.Close()

	form := url.Values{
		"yes": {"ok"},
	}
	resp, err := http.Get(s.URL + "?" + form.Encode())

	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
