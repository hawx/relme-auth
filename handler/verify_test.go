package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
)

type fakeSessionStore struct {
	Session data.Session
}

func (s *fakeSessionStore) Save(session *data.Session) {}

func (s *fakeSessionStore) Get(id string) (data.Session, bool) {
	return s.Session, true
}

func (s *fakeSessionStore) GetByCode(code string) (data.Session, bool) {
	if code == s.Session.Code {
		return s.Session, true
	}

	return data.Session{}, false
}

func TestVerify(t *testing.T) {
	assert := assert.New(t)

	session := data.Session{
		ClientID:    "http://client.example.com",
		RedirectURI: "http://done.example.com",
		Me:          "it is me",
		CreatedAt:   time.Now(),
		Code:        "1234",
	}

	s := httptest.NewServer(Verify(&fakeSessionStore{Session: session}))
	defer s.Close()

	form := url.Values{"code": {session.Code}, "client_id": {session.ClientID}, "redirect_uri": {session.RedirectURI}}
	resp, err := http.PostForm(s.URL, form)
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	var v struct {
		Me string `json:"me"`
	}
	json.NewDecoder(resp.Body).Decode(&v)
	assert.Equal(v.Me, session.Me)
}

func TestVerifyWithBadForm(t *testing.T) {
	assert := assert.New(t)

	session := data.Session{
		ClientID:    "http://client.example.com",
		RedirectURI: "http://done.example.com",
		Me:          "it is me",
		CreatedAt:   time.Now(),
		Code:        "1234",
	}

	s := httptest.NewServer(Verify(&fakeSessionStore{Session: session}))
	defer s.Close()

	testCases := map[string]url.Values{
		"missing code":           url.Values{"client_id": {"http://example.com"}, "redirect_uri": {"http://example.com"}},
		"missing client_id":      url.Values{"code": {"123"}, "redirect_uri": {"http://example.com"}},
		"missing redirect_uri":   url.Values{"code": {"123"}, "client_id": {"http://example.com"}},
		"incorrect code":         url.Values{"code": {"9876"}, "client_id": {session.ClientID}, "redirect_uri": {session.RedirectURI}},
		"incorrect client_id":    url.Values{"code": {session.Code}, "client_id": {"what"}, "redirect_uri": {session.RedirectURI}},
		"incorrect redirect_uri": url.Values{"code": {session.Code}, "client_id": {session.ClientID}, "redirect_uri": {"what"}},
	}

	for name, form := range testCases {
		t.Run(name, func(t *testing.T) {
			resp, err := http.PostForm(s.URL, form)
			assert.Nil(err)
			assert.Equal(http.StatusBadRequest, resp.StatusCode)

			var v struct {
				Error string `json:"error"`
			}
			json.NewDecoder(resp.Body).Decode(&v)
			assert.Equal(v.Error, "invalid_request")
		})
	}
}
