package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
)

func TestCallback(t *testing.T) {
	store := &fakeSessionStore{
		Session: data.Session{
			Me:          "me",
			Code:        "my-code",
			State:       "my-state",
			RedirectURI: "http://example.com/callback",
		},
	}

	s := httptest.NewServer(Callback(store, &fakeStrategy{}))
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
	s := httptest.NewServer(Callback(&fakeSessionStore{}, &fakeStrategy{}))
	defer s.Close()

	form := url.Values{
		"yes": {"ok"},
	}
	resp, err := http.Get(s.URL + "?" + form.Encode())

	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCallbackWhenProviderSaysTheyAreUnauthorized(t *testing.T) {
	store := &fakeSessionStore{
		Session: data.Session{
			Me:          "me",
			Code:        "my-code",
			State:       "my-state",
			RedirectURI: "http://example.com/callback",
		},
	}

	s := httptest.NewServer(Callback(store, &unauthorizedStrategy{}))
	defer s.Close()

	form := url.Values{
		"yes": {"ok"},
	}
	resp, err := http.Get(s.URL + "?" + form.Encode())

	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCallbackWhenProviderErrors(t *testing.T) {
	store := &fakeSessionStore{
		Session: data.Session{
			Me:          "me",
			Code:        "my-code",
			State:       "my-state",
			RedirectURI: "http://example.com/callback",
		},
	}

	s := httptest.NewServer(Callback(store, &errorStrategy{}))
	defer s.Close()

	form := url.Values{
		"yes": {"ok"},
	}
	resp, err := http.Get(s.URL + "?" + form.Encode())

	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
