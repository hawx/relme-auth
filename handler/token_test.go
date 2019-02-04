package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
)

func TestToken(t *testing.T) {
	assert := assert.New(t)

	session := data.Session{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		Code:         "1234",
		ResponseType: "code",
		Token:        "abcde",
		Scopes:       []string{"create", "update"},
	}

	s := httptest.NewServer(Token(&fakeSessionStore{Session: session}))
	defer s.Close()

	resp, err := http.PostForm(s.URL, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {session.Code},
		"client_id":    {session.ClientID},
		"redirect_uri": {session.RedirectURI},
		"me":           {session.Me},
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
	assert.Equal(session.Token, v.AccessToken)
	assert.Equal("Bearer", v.TokenType)
	assert.Equal(strings.Join(session.Scopes, " "), v.Scope)
	assert.Equal(session.Me, v.Me)
}

func TestTokenWithBadParams(t *testing.T) {
	session := data.Session{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		Code:         "1234",
		ResponseType: "code",
		Token:        "abcde",
		Scopes:       []string{"create", "update"},
	}

	s := httptest.NewServer(Token(&fakeSessionStore{Session: session}))
	defer s.Close()

	testCases := map[string]url.Values{
		"missing grant type": url.Values{
			"code":         {session.Code},
			"client_id":    {session.ClientID},
			"redirect_uri": {session.RedirectURI},
			"me":           {session.Me},
		},
		"unknown grant type": url.Values{
			"grant_type":   {"what"},
			"code":         {session.Code},
			"client_id":    {session.ClientID},
			"redirect_uri": {session.RedirectURI},
			"me":           {session.Me},
		},
		"invalid code": url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {"nope"},
			"client_id":    {session.ClientID},
			"redirect_uri": {session.RedirectURI},
			"me":           {session.Me},
		},
		"mismatched clientID": url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {session.Code},
			"client_id":    {"nope"},
			"redirect_uri": {session.RedirectURI},
			"me":           {session.Me},
		},
		"mismatched redirectURI": url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {session.Code},
			"client_id":    {session.ClientID},
			"redirect_uri": {"nope"},
			"me":           {session.Me},
		},
		"mismatched me": url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {session.Code},
			"client_id":    {session.ClientID},
			"redirect_uri": {session.RedirectURI},
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
	session := data.Session{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now().Add(-60 * time.Second),
		Code:         "1234",
		ResponseType: "code",
		Token:        "abcde",
		Scopes:       []string{"create", "update"},
	}

	s := httptest.NewServer(Token(&fakeSessionStore{Session: session}))
	defer s.Close()

	resp, err := http.PostForm(s.URL, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {session.Code},
		"client_id":    {session.ClientID},
		"redirect_uri": {session.RedirectURI},
		"me":           {session.Me},
	})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRevokeToken(t *testing.T) {
	assert := assert.New(t)

	session := data.Session{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		Code:         "1234",
		ResponseType: "code",
		Token:        "abcde",
		Scopes:       []string{"create", "update"},
	}
	sessionStore := &fakeSessionStore{Session: session}

	s := httptest.NewServer(Token(sessionStore))
	defer s.Close()

	resp, err := http.PostForm(s.URL, url.Values{
		"action": {"revoke"},
		"token":  {session.Token},
	})
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	assert.Equal(sessionStore.Session, data.Session{})
}

func TestVerifyToken(t *testing.T) {
	assert := assert.New(t)

	session := data.Session{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		Code:         "1234",
		ResponseType: "code",
		Token:        "abcde",
		Scopes:       []string{"create", "update"},
	}

	s := httptest.NewServer(Token(&fakeSessionStore{Session: session}))
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)
	req.Header.Add("Authorization", "Bearer "+session.Token)

	resp, err := http.DefaultClient.Do(req)
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	var v struct {
		Me       string `json:"me"`
		ClientID string `json:"client_id"`
		Scope    string `json:"scope"`
	}
	assert.Nil(json.NewDecoder(resp.Body).Decode(&v))
	assert.Equal(session.Me, v.Me)
	assert.Equal(session.ClientID, v.ClientID)
	assert.Equal(strings.Join(session.Scopes, " "), v.Scope)
}

func TestVerifyTokenWithBadParams(t *testing.T) {
	session := data.Session{
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://done.example.com",
		Me:           "it is me",
		CreatedAt:    time.Now(),
		Code:         "1234",
		ResponseType: "code",
		Token:        "abcde",
		Scopes:       []string{"create", "update"},
	}

	s := httptest.NewServer(Token(&fakeSessionStore{Session: session}))
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)
	req.Header.Add("Authorization", "Bearer "+session.Token)

	testCases := map[string]string{
		"invalid auth header": "one-part",
		"not bearer":          "something " + session.Token,
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
