package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/internal/data"
)

func codeGenerator() (string, error) { return "my-code", nil }

type fakeCallbackStore struct {
	session data.Session
	code    data.Code
}

func (s *fakeCallbackStore) SaveLogin(w http.ResponseWriter, r *http.Request, me string) error {
	return nil
}

func (s *fakeCallbackStore) Session(me string) (data.Session, error) {
	if me == s.session.Me {
		return s.session, nil
	}

	return data.Session{}, errors.New("what")
}

func (s *fakeCallbackStore) CreateCode(me, code string, createdAt time.Time) error {
	if me == s.session.Me {
		s.code = data.Code{
			Code:         code,
			ResponseType: s.session.ResponseType,
			Me:           s.session.Me,
			ClientID:     s.session.ClientID,
			RedirectURI:  s.session.RedirectURI,
			Scope:        s.session.Scope,
			CreatedAt:    createdAt,
			ExpiresAt:    createdAt.Add(time.Minute),
		}
		return nil
	}
	return errors.New("who")
}

func TestCallback(t *testing.T) {
	store := &fakeCallbackStore{
		session: data.Session{
			Me:          "me",
			State:       "my-state",
			RedirectURI: "http://example.com/callback",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(5 * time.Minute),
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

func TestCallbackWhenSessionExpired(t *testing.T) {
	store := &fakeCallbackStore{
		session: data.Session{
			Me:          "me",
			State:       "my-state",
			RedirectURI: "http://example.com/callback",
			CreatedAt:   time.Now().Add(-10 * time.Minute),
			ExpiresAt:   time.Now().Add(-10 * time.Minute),
		},
	}

	s := httptest.NewServer(Callback(store, &fakeStrategy{}, codeGenerator))
	defer s.Close()

	form := url.Values{
		"yes": {"ok"},
	}
	resp, err := http.Get(s.URL + "?" + form.Encode())

	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "", resp.Header.Get("Location"))
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
