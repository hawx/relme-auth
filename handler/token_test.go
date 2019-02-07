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

func fakeGenerator() string { return "a token" }

type fakeTokenStore struct {
	code  data.Code
	token data.Token
}

func (s *fakeTokenStore) Code(code string) (data.Code, error) {
	if code == s.code.Code {
		return s.code, nil
	}
	return data.Code{}, errors.New("no")
}

func (s *fakeTokenStore) Token(t string) (data.Token, error) {
	if t == s.token.Token {
		return s.token, nil
	}
	return data.Token{}, errors.New("no")
}

func (s *fakeTokenStore) CreateToken(t data.Token) error {
	s.token = t
	return nil
}

func (s *fakeTokenStore) RevokeToken(t string) error {
	s.token = data.Token{}
	return nil
}

func TestToken(t *testing.T) {
	assert := assert.New(t)

	code := data.Code{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		Code:         "1234",
		ResponseType: "code",
		Scope:        "create update",
	}

	s := httptest.NewServer(Token(&fakeTokenStore{code: code}, fakeGenerator))
	defer s.Close()

	resp, err := http.PostForm(s.URL, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code.Code},
		"client_id":    {code.ClientID},
		"redirect_uri": {code.RedirectURI},
		"me":           {code.Me},
	})
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	var v struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Me          string `json:"me"`
	}
	assert.Nil(json.NewDecoder(resp.Body).Decode(&v))
	assert.Equal("a token", v.AccessToken)
	assert.Equal("Bearer", v.TokenType)
	assert.Equal(code.Scope, v.Scope)
	assert.Equal(code.Me, v.Me)
}

func TestTokenWithBadParams(t *testing.T) {
	code := data.Code{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		Code:         "1234",
		ResponseType: "code",
		Scope:        "create update",
	}

	s := httptest.NewServer(Token(&fakeTokenStore{code: code}, fakeGenerator))
	defer s.Close()

	testCases := map[string]url.Values{
		"missing grant type": url.Values{
			"code":         {code.Code},
			"client_id":    {code.ClientID},
			"redirect_uri": {code.RedirectURI},
			"me":           {code.Me},
		},
		"unknown grant type": url.Values{
			"grant_type":   {"what"},
			"code":         {code.Code},
			"client_id":    {code.ClientID},
			"redirect_uri": {code.RedirectURI},
			"me":           {code.Me},
		},
		"invalid code": url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {"nope"},
			"client_id":    {code.ClientID},
			"redirect_uri": {code.RedirectURI},
			"me":           {code.Me},
		},
		"mismatched clientID": url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {code.Code},
			"client_id":    {"nope"},
			"redirect_uri": {code.RedirectURI},
			"me":           {code.Me},
		},
		"mismatched redirectURI": url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {code.Code},
			"client_id":    {code.ClientID},
			"redirect_uri": {"nope"},
			"me":           {code.Me},
		},
		"mismatched me": url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {code.Code},
			"client_id":    {code.ClientID},
			"redirect_uri": {code.RedirectURI},
			"me":           {"nope"},
		},
	}

	for name, form := range testCases {
		t.Run(name, func(t *testing.T) {
			resp, err := http.PostForm(s.URL, form)
			assert.Nil(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestTokenWithExpiredSession(t *testing.T) {
	code := data.Code{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now().Add(-60 * time.Second),
		Code:         "1234",
		ResponseType: "code",
		Scope:        "create update",
	}

	s := httptest.NewServer(Token(&fakeTokenStore{code: code}, fakeGenerator))
	defer s.Close()

	resp, err := http.PostForm(s.URL, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code.Code},
		"client_id":    {code.ClientID},
		"redirect_uri": {code.RedirectURI},
		"me":           {code.Me},
	})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRevokeToken(t *testing.T) {
	assert := assert.New(t)

	token := data.Token{
		Token:     "abcde",
		ClientID:  "http://client.example.com",
		Scope:     "create update",
		Me:        "it is me",
		CreatedAt: time.Now(),
	}
	sessionStore := &fakeTokenStore{token: token}

	s := httptest.NewServer(Token(sessionStore, fakeGenerator))
	defer s.Close()

	resp, err := http.PostForm(s.URL, url.Values{
		"action": {"revoke"},
		"token":  {token.Token},
	})
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	assert.Equal(sessionStore.token, data.Token{})
}

func TestVerifyToken(t *testing.T) {
	assert := assert.New(t)

	token := data.Token{
		Token:     "abcde",
		ClientID:  "http://client.example.com",
		Scope:     "create update",
		Me:        "it is me",
		CreatedAt: time.Now(),
	}

	s := httptest.NewServer(Token(&fakeTokenStore{token: token}, fakeGenerator))
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)
	req.Header.Add("Authorization", "Bearer "+token.Token)

	resp, err := http.DefaultClient.Do(req)
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	var v struct {
		Me       string `json:"me"`
		ClientID string `json:"client_id"`
		Scope    string `json:"scope"`
	}
	assert.Nil(json.NewDecoder(resp.Body).Decode(&v))
	assert.Equal(token.Me, v.Me)
	assert.Equal(token.ClientID, v.ClientID)
	assert.Equal(token.Scope, v.Scope)
}

func TestVerifyTokenWithBadParams(t *testing.T) {
	token := data.Token{
		ClientID:  "http://client.example.com",
		Me:        "it is me",
		CreatedAt: time.Now(),
		Token:     "abcde",
		Scope:     "create update",
	}

	s := httptest.NewServer(Token(&fakeTokenStore{token: token}, fakeGenerator))
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)
	req.Header.Add("Authorization", "Bearer "+token.Token)

	testCases := map[string]string{
		"invalid auth header": "one-part",
		"not bearer":          "something " + token.Token,
		"unknown token":       "Bearer what",
	}

	for name, header := range testCases {
		t.Run(name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", s.URL, nil)
			req.Header.Add("Authorization", "Bearer "+header)

			resp, err := http.DefaultClient.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}
