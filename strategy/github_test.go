package strategy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data/memory"
)

func TestGitHubMatch(t *testing.T) {
	gitHub := GitHub(memory.New(), id, secret)

	testCases := []string{
		"https://github.com/somebody",
		"http://github.com/somebody",
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			parsed, err := url.Parse(tc)
			assert.Nil(t, err)
			assert.True(t, gitHub.Match(parsed))
		})
	}
}

func TestGitHubNotMatch(t *testing.T) {
	gitHub := GitHub(memory.New(), id, secret)

	testCases := []string{
		"https://www.github.com/somebody",
		"github.com/somebody",
		"https://githubz.com/somebody",
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			parsed, err := url.Parse(tc)
			assert.Nil(t, err)
			assert.False(t, gitHub.Match(parsed))
		})
	}
}

func TestGitHubAuthFlow(t *testing.T) {
	const (
		expectedURL = "http://whatever.example.com"
		state       = "randomstatestring"
		code        = "somecode"
		accessToken = "the-access-key"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "POST" && r.URL.Path == "/oauth/access_token" &&
			r.PostFormValue("client_id") == id &&
			r.PostFormValue("code") == code {

			w.Write([]byte(url.Values{
				"access_token": {accessToken},
			}.Encode()))
		}

		if r.Method == "GET" && r.URL.Path == "/user" &&
			r.Header.Get("Authorization") == "Bearer "+accessToken {

			json.NewEncoder(w).Encode(map[string]interface{}{
				"blog": expectedURL,
			})
		}
	}))
	defer server.Close()

	gitHub := &authGitHub{
		Conf: &oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Scopes:       []string{},
			Endpoint: oauth2.Endpoint{
				AuthURL:  server.URL + "/oauth/authorize",
				TokenURL: server.URL + "/oauth/access_token",
			},
		},
		Store:  &oneStore{State: state},
		ApiURI: server.URL,
	}

	expectedRedirectURL := fmt.Sprintf("%s/oauth/authorize?access_type=offline&client_id=%s&response_type=code&state=%s", server.URL, id, state)

	// 1. Redirect
	redirectURL, err := gitHub.Redirect(expectedURL)
	assert.Nil(t, err)
	assert.Equal(t, expectedRedirectURL, redirectURL)

	// 2. Callback
	profileURL, err := gitHub.Callback(url.Values{
		"state": {state},
		"code":  {code},
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedURL, profileURL)
}

type oneStore struct {
	State string
	Link  string
}

func (s *oneStore) Insert(link string) (state string, err error) {
	s.Link = link

	return s.State, nil
}

func (s *oneStore) Set(key, value string) error {
	return errors.New("not used")
}

func (s *oneStore) Claim(state string) (link string, ok bool) {
	if state != s.State {
		return "", false
	}

	return s.Link, true
}
