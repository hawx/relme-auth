package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
)

type fakeVerifyStore struct {
	code data.Code
}

func (s fakeVerifyStore) Code(code string) (data.Code, error) {
	if code == s.code.Code {
		return s.code, nil
	}

	return data.Code{}, errors.New("hey")
}

func TestVerify(t *testing.T) {
	assert := assert.Wrap(t)

	code := data.Code{
		ClientID:     "http://client.example.com/",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Minute),
		Code:         "1234",
		ResponseType: "id",
	}

	s := httptest.NewServer(Verify(&fakeVerifyStore{code: code}))
	defer s.Close()

	form := url.Values{"code": {code.Code}, "client_id": {code.ClientID}, "redirect_uri": {code.RedirectURI}}
	resp, err := http.PostForm(s.URL, form)
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusOK)

	var v struct {
		Me string `json:"me"`
	}
	json.NewDecoder(resp.Body).Decode(&v)
	assert(v.Me).Equal(code.Me)
}

func TestVerifyWithExpiredSession(t *testing.T) {
	assert := assert.Wrap(t)

	code := data.Code{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now().Add(-60 * time.Second),
		ExpiresAt:    time.Now().Add(-time.Second),
		Code:         "1234",
		ResponseType: "id",
	}

	s := httptest.NewServer(Verify(&fakeVerifyStore{code: code}))
	defer s.Close()

	form := url.Values{"code": {code.Code}, "client_id": {code.ClientID}, "redirect_uri": {code.RedirectURI}}
	resp, err := http.PostForm(s.URL, form)
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusBadRequest)

	var v struct {
		Error string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&v)
	assert(v.Error).Equal("invalid_request")
}

func TestVerifyWithBadForm(t *testing.T) {
	code := data.Code{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Minute),
		Code:         "1234",
		ResponseType: "id",
	}

	s := httptest.NewServer(Verify(&fakeVerifyStore{code: code}))
	defer s.Close()

	testCases := map[string]url.Values{
		"missing code":           url.Values{"client_id": {"http://example.com"}, "redirect_uri": {"http://example.com"}},
		"missing client_id":      url.Values{"code": {"123"}, "redirect_uri": {"http://example.com"}},
		"missing redirect_uri":   url.Values{"code": {"123"}, "client_id": {"http://example.com"}},
		"incorrect code":         url.Values{"code": {"9876"}, "client_id": {code.ClientID}, "redirect_uri": {code.RedirectURI}},
		"incorrect client_id":    url.Values{"code": {code.Code}, "client_id": {"what"}, "redirect_uri": {code.RedirectURI}},
		"incorrect redirect_uri": url.Values{"code": {code.Code}, "client_id": {code.ClientID}, "redirect_uri": {"what"}},
	}

	for name, form := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.Wrap(t)

			resp, err := http.PostForm(s.URL, form)
			assert(err).Must.Nil()
			assert(resp.StatusCode).Equal(http.StatusBadRequest)

			var v struct {
				Error string `json:"error"`
			}
			json.NewDecoder(resp.Body).Decode(&v)
			assert(v.Error).Equal("invalid_request")
		})
	}
}

func TestVerifyWithCodeSession(t *testing.T) {
	assert := assert.Wrap(t)

	code := data.Code{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Minute),
		Code:         "1234",
		ResponseType: "code",
	}

	s := httptest.NewServer(Verify(&fakeVerifyStore{code: code}))
	defer s.Close()

	form := url.Values{"code": {code.Code}, "client_id": {code.ClientID}, "redirect_uri": {code.RedirectURI}}
	resp, err := http.PostForm(s.URL, form)
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusBadRequest)

	var v struct {
		Error string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&v)
	assert(v.Error).Equal("invalid_request")
}
