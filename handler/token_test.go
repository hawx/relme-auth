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

func fakeGenerator() (string, error) { return "a token", nil }

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
	assert := assert.Wrap(t)

	code := data.Code{
		ClientID:     "http://client.example.com/",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Minute),
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
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusOK)
	assert(resp.Header.Get("Content-Type")).Equal("application/json")

	var v struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Me          string `json:"me"`
	}
	assert(json.NewDecoder(resp.Body).Decode(&v)).Must.Nil()
	assert(v.AccessToken).Equal("a token")
	assert(v.TokenType).Equal("Bearer")
	assert(v.Scope).Equal(code.Scope)
	assert(v.Me).Equal(code.Me)
}

func TestTokenWithPKCE(t *testing.T) {
	assert := assert.Wrap(t)

	code := data.Code{
		ClientID:            "http://client.example.com/",
		RedirectURI:         "http://done.example.com",
		Me:                  "it is me",
		CreatedAt:           time.Now(),
		ExpiresAt:           time.Now().Add(time.Minute),
		CodeChallenge:       "pixies",
		CodeChallengeMethod: "plain",
		Code:                "1234",
		ResponseType:        "code",
		Scope:               "create update",
	}

	s := httptest.NewServer(Token(&fakeTokenStore{code: code}, fakeGenerator))
	defer s.Close()

	resp, err := http.PostForm(s.URL, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code.Code},
		"client_id":     {code.ClientID},
		"redirect_uri":  {code.RedirectURI},
		"me":            {code.Me},
		"code_verifier": {code.CodeChallenge},
	})
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusOK)
	assert(resp.Header.Get("Content-Type")).Equal("application/json")

	var v struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Me          string `json:"me"`
	}
	assert(json.NewDecoder(resp.Body).Decode(&v)).Must.Nil()
	assert(v.AccessToken).Equal("a token")
	assert(v.TokenType).Equal("Bearer")
	assert(v.Scope).Equal(code.Scope)
	assert(v.Me).Equal(code.Me)
}

func TestTokenWithBadParams(t *testing.T) {
	code := data.Code{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Minute),
		Code:         "1234",
		ResponseType: "code",
		Scope:        "create update",
	}

	s := httptest.NewServer(Token(&fakeTokenStore{code: code}, fakeGenerator))
	defer s.Close()

	testCases := map[string]url.Values{
		"missing grant type": {
			"code":         {code.Code},
			"client_id":    {code.ClientID},
			"redirect_uri": {code.RedirectURI},
			"me":           {code.Me},
		},
		"unknown grant type": {
			"grant_type":   {"what"},
			"code":         {code.Code},
			"client_id":    {code.ClientID},
			"redirect_uri": {code.RedirectURI},
			"me":           {code.Me},
		},
		"invalid code": {
			"grant_type":   {"authorization_code"},
			"code":         {"nope"},
			"client_id":    {code.ClientID},
			"redirect_uri": {code.RedirectURI},
			"me":           {code.Me},
		},
		"mismatched clientID": {
			"grant_type":   {"authorization_code"},
			"code":         {code.Code},
			"client_id":    {"nope"},
			"redirect_uri": {code.RedirectURI},
			"me":           {code.Me},
		},
		"mismatched redirectURI": {
			"grant_type":   {"authorization_code"},
			"code":         {code.Code},
			"client_id":    {code.ClientID},
			"redirect_uri": {"nope"},
			"me":           {code.Me},
		},
		"mismatched me": {
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

func TestTokenWithBadPKCEParams(t *testing.T) {
	code := data.Code{
		ClientID:            "http://client.example.com/",
		RedirectURI:         "http://done.example.com",
		Me:                  "it is me",
		CreatedAt:           time.Now(),
		ExpiresAt:           time.Now().Add(time.Minute),
		CodeChallenge:       "pixies",
		CodeChallengeMethod: "plain",
		Code:                "1234",
		ResponseType:        "code",
		Scope:               "create update",
	}

	s := httptest.NewServer(Token(&fakeTokenStore{code: code}, fakeGenerator))
	defer s.Close()

	testCases := map[string]url.Values{
		"incorrect code verifier": {
			"grant_type":    {"authorization_code"},
			"code":          {code.Code},
			"client_id":     {code.ClientID},
			"redirect_uri":  {code.RedirectURI},
			"me":            {code.Me},
			"code_verifier": {"nope"},
		},
		"missing code verifier": {
			"grant_type":   {"authorization_code"},
			"code":         {code.Code},
			"client_id":    {code.ClientID},
			"redirect_uri": {code.RedirectURI},
			"me":           {code.Me},
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

func TestTokenWithUnexpectedPKCEParams(t *testing.T) {
	code := data.Code{
		ClientID:     "http://client.example.com/",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Minute),
		Code:         "1234",
		ResponseType: "code",
		Scope:        "create update",
	}

	s := httptest.NewServer(Token(&fakeTokenStore{code: code}, fakeGenerator))
	defer s.Close()

	resp, err := http.PostForm(s.URL, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code.Code},
		"client_id":     {code.ClientID},
		"redirect_uri":  {code.RedirectURI},
		"me":            {code.Me},
		"code_verifier": {"nope"},
	})

	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTokenWithExpiredSession(t *testing.T) {
	code := data.Code{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now().Add(-60 * time.Second),
		ExpiresAt:    time.Now().Add(-time.Second),
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
	assert := assert.Wrap(t)

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
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusOK)

	assert(sessionStore.token).Equal(data.Token{})
}

func TestVerifyToken(t *testing.T) {
	assert := assert.Wrap(t)

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
	assert(err).Must.Nil()
	assert(resp.StatusCode).Equal(http.StatusOK)
	assert(resp.Header.Get("Content-Type")).Equal("application/json")

	var v struct {
		Me       string `json:"me"`
		ClientID string `json:"client_id"`
		Scope    string `json:"scope"`
	}
	assert(json.NewDecoder(resp.Body).Decode(&v)).Must.Nil()
	assert(v.Me).Equal(token.Me)
	assert(v.ClientID).Equal(token.ClientID)
	assert(v.Scope).Equal(token.Scope)
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
