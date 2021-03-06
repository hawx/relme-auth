package strategy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/garyburd/go-oauth/oauth"
	"hawx.me/code/assert"
)

func TestTwitterMatch(t *testing.T) {
	twitter := Twitter("", new(fakeStore), id, secret, http.DefaultClient)

	testCases := []string{
		"https://twitter.com/somebody",
		"http://twitter.com/somebody",
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			parsed, err := url.Parse(tc)
			assert.Nil(t, err)
			assert.True(t, twitter.Match(parsed))
		})
	}
}

func TestTwitterNotMatch(t *testing.T) {
	twitter := Twitter("", new(fakeStore), id, secret, http.DefaultClient)

	testCases := []string{
		"https://www.twitter.com/somebody",
		"twitter.com/somebody",
		"https://twitterz.com/somebody",
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			parsed, err := url.Parse(tc)
			assert.Nil(t, err)
			assert.False(t, twitter.Match(parsed))
		})
	}
}

func TestTwitterAuthFlow(t *testing.T) {
	const (
		expectedURL = "http://whatever.example.com"
		shortURL    = "https://t.co/whatever"
		tempToken   = "temp-token"
		tempSecret  = "temp-secret"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/oauth/request_token" &&
			hasParam(r, "oauth_consumer_key", id) {

			w.Write([]byte(url.Values{
				"oauth_callback_confirmed": {"true"},
				"oauth_token":              {tempToken},
				"oauth_token_secret":       {tempSecret},
			}.Encode()))
		}

		if r.Method == "POST" && r.URL.Path == "/oauth/access_token" &&
			hasParam(r, "oauth_token", tempToken) &&
			hasParam(r, "oauth_verifier", tempSecret) &&
			hasParam(r, "oauth_consumer_key", id) {

			w.Write([]byte(url.Values{
				"oauth_token":        {tempToken},
				"oauth_token_secret": {tempSecret},
			}.Encode()))
		}

		if r.Method == "GET" && r.URL.Path == "/1.1/account/verify_credentials.json" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"url": shortURL,
				"entities": map[string]interface{}{
					"url": map[string]interface{}{
						"urls": []map[string]interface{}{
							{
								"url":          shortURL,
								"expanded_url": expectedURL,
							},
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	twitter := &authTwitter{
		client: oauth.Client{
			TemporaryCredentialRequestURI: server.URL + "/oauth/request_token",
			ResourceOwnerAuthorizationURI: server.URL + "/oauth/authorize",
			TokenRequestURI:               server.URL + "/oauth/access_token",
			Credentials: oauth.Credentials{
				Token:  id,
				Secret: secret,
			},
		},
		callbackURL: "",
		store:       new(fakeStore),
		apiURI:      server.URL + "/1.1",
		httpClient:  http.DefaultClient,
	}

	expectedRedirectURL := fmt.Sprintf("%s/oauth/authorize?oauth_token=%s", server.URL, tempToken)

	// 1. Redirect
	redirectURL, err := twitter.Redirect(expectedURL, "")
	assert.Nil(t, err)
	assert.Equal(t, expectedRedirectURL, redirectURL)

	// 2. Callback
	profileURL, err := twitter.Callback(url.Values{
		"oauth_token":    {tempToken},
		"oauth_verifier": {tempSecret},
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedURL, profileURL)
}

func TestTwitterAuthFlowWithBadUser(t *testing.T) {
	const (
		expectedURL = "http://whatever.example.com"
		shortURL    = "https://t.co/whatever"
		tempToken   = "temp-token"
		tempSecret  = "temp-secret"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/oauth/request_token" &&
			hasParam(r, "oauth_consumer_key", id) {

			w.Write([]byte(url.Values{
				"oauth_callback_confirmed": {"true"},
				"oauth_token":              {tempToken},
				"oauth_token_secret":       {tempSecret},
			}.Encode()))
		}

		if r.Method == "POST" && r.URL.Path == "/oauth/access_token" &&
			hasParam(r, "oauth_token", tempToken) &&
			hasParam(r, "oauth_verifier", tempSecret) &&
			hasParam(r, "oauth_consumer_key", id) {

			w.Write([]byte(url.Values{
				"oauth_token":        {tempToken},
				"oauth_token_secret": {tempSecret},
			}.Encode()))
		}

		if r.Method == "GET" && r.URL.Path == "/1.1/account/verify_credentials.json" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"url": "nope",
				"entities": map[string]interface{}{
					"url": map[string]interface{}{
						"urls": []map[string]interface{}{
							{
								"url":          "nope",
								"expanded_url": "nope",
							},
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	twitter := &authTwitter{
		client: oauth.Client{
			TemporaryCredentialRequestURI: server.URL + "/oauth/request_token",
			ResourceOwnerAuthorizationURI: server.URL + "/oauth/authorize",
			TokenRequestURI:               server.URL + "/oauth/access_token",
			Credentials: oauth.Credentials{
				Token:  id,
				Secret: secret,
			},
		},
		callbackURL: "",
		store:       new(fakeStore),
		apiURI:      server.URL + "/1.1",
		httpClient:  http.DefaultClient,
	}

	expectedRedirectURL := fmt.Sprintf("%s/oauth/authorize?oauth_token=%s", server.URL, tempToken)

	// 1. Redirect
	redirectURL, err := twitter.Redirect(expectedURL, "")
	assert.Nil(t, err)
	assert.Equal(t, expectedRedirectURL, redirectURL)

	// 2. Callback
	profileURL, err := twitter.Callback(url.Values{
		"oauth_token":    {tempToken},
		"oauth_verifier": {tempSecret},
	})
	assert.Equal(t, ErrUnauthorized, err)
	assert.Equal(t, "", profileURL)
}
