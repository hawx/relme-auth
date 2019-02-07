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

const (
	id     = "my-cool-id"
	secret = "my-cool-secret"
)

func TestFlickrMatch(t *testing.T) {
	flickr := Flickr("", new(fakeStore), id, secret, http.DefaultClient)

	testCases := []string{
		"https://www.flickr.com/somebody",
		"http://www.flickr.com/somebody",
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			parsed, err := url.Parse(tc)
			assert.Nil(t, err)
			assert.True(t, flickr.Match(parsed))
		})
	}
}

func TestFlickrNotMatch(t *testing.T) {
	flickr := Flickr("", new(fakeStore), id, secret, http.DefaultClient)

	testCases := []string{
		"https://www.flickrz.com/somebody",
		"www.flickr.com/somebody",
		"https://flickr.com/somebody",
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			parsed, err := url.Parse(tc)
			assert.Nil(t, err)
			assert.False(t, flickr.Match(parsed))
		})
	}
}

func TestFlickrAuthFlow(t *testing.T) {
	const (
		expectedURL = "http://whatever.example.com"
		tempToken   = "temp-token"
		tempSecret  = "temp-secret"
		nsid        = "someflickrnsid"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/oauth/request_token" &&
			r.PostFormValue("oauth_consumer_key") == id {

			w.Write([]byte(url.Values{
				"oauth_callback_confirmed": {"true"},
				"oauth_token":              {tempToken},
				"oauth_token_secret":       {tempSecret},
			}.Encode()))
		}

		if r.Method == "POST" && r.URL.Path == "/oauth/access_token" &&
			r.PostFormValue("oauth_token") == tempToken &&
			r.PostFormValue("oauth_verifier") == tempSecret &&
			r.PostFormValue("oauth_consumer_key") == id {

			w.Write([]byte(url.Values{
				"fullname":           {"Someone Someone"},
				"oauth_token":        {tempToken},
				"oauth_token_secret": {tempSecret},
				"user_nsid":          {nsid},
				"username":           {"someone"},
			}.Encode()))
		}

		if r.Method == "GET" && r.URL.Path == "/services/rest" &&
			r.FormValue("nojsoncallback") == "1" &&
			r.FormValue("format") == "json" &&
			r.FormValue("api_key") == id &&
			r.FormValue("user_id") == nsid &&
			r.FormValue("method") == "flickr.profile.getProfile" {

			json.NewEncoder(w).Encode(map[string]interface{}{
				"profile": map[string]string{
					"website": expectedURL,
				},
			})
		}
	}))
	defer server.Close()

	flickr := &authFlickr{
		Client: oauth.Client{
			TemporaryCredentialRequestURI: server.URL + "/oauth/request_token",
			ResourceOwnerAuthorizationURI: server.URL + "/oauth/authorize",
			TokenRequestURI:               server.URL + "/oauth/access_token",
			Credentials: oauth.Credentials{
				Token:  id,
				Secret: secret,
			},
		},
		CallbackURL: "",
		Store:       new(fakeStore),
		APIKey:      id,
		APIURI:      server.URL + "/services/rest",
		httpClient:  http.DefaultClient,
	}

	expectedRedirectURL := fmt.Sprintf("%s/oauth/authorize?oauth_token=%s&perms=read", server.URL, tempToken)

	// 1. Redirect
	redirectURL, err := flickr.Redirect(expectedURL)
	assert.Nil(t, err)
	assert.Equal(t, expectedRedirectURL, redirectURL)

	// 2. Callback
	profileURL, err := flickr.Callback(url.Values{
		"oauth_token":    {tempToken},
		"oauth_verifier": {tempSecret},
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedURL, profileURL)
}
