package strategy

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"hawx.me/code/relme-auth/state"

	"github.com/garyburd/go-oauth/oauth"
)

type authTwitter struct {
	Client      oauth.Client
	CallbackURL string
	Store       state.Store
}

func Twitter(store state.Store, id, secret string) Strategy {
	oauthClient := oauth.Client{
		TemporaryCredentialRequestURI: "https://api.twitter.com/oauth/request_token",
		ResourceOwnerAuthorizationURI: "https://api.twitter.com/oauth/authenticate",
		TokenRequestURI:               "https://api.twitter.com/oauth/access_token",
		Credentials: oauth.Credentials{
			Token:  id,
			Secret: secret,
		},
	}

	return &authTwitter{
		Client:      oauthClient,
		CallbackURL: "http://localhost:8080/oauth/callback/twitter",
		Store:       store,
	}
}

func (strategy *authTwitter) Match(me *url.URL) bool {
	return me.Hostname() == "twitter.com"
}

func (strategy *authTwitter) Redirect(expectedURL string) (redirectURL string, err error) {
	tempCred, err := strategy.Client.RequestTemporaryCredentials(http.DefaultClient, strategy.CallbackURL, nil)
	if err != nil {
		return "", err
	}

	// these are temporary hacks
	if err := strategy.Store.Set(tempCred.Token, tempCred.Secret); err != nil {
		return "", err
	}
	if err := strategy.Store.Set(tempCred.Secret, expectedURL); err != nil {
		return "", err
	}

	return strategy.Client.AuthorizationURL(tempCred, nil), nil
}

func (strategy *authTwitter) Callback(form url.Values) (string, error) {
	oauthToken := form.Get("oauth_token")
	expectedSecret, ok := strategy.Store.Claim(oauthToken)
	if !ok {
		return "", errors.New("Unknown oauth_token")
	}

	tempCred := &oauth.Credentials{
		Token:  oauthToken,
		Secret: expectedSecret,
	}
	tokenCred, _, err := strategy.Client.RequestToken(http.DefaultClient, tempCred, form.Get("oauth_verifier"))
	if err != nil {
		return "", errors.New("Error getting request token, " + err.Error())
	}

	resp, err := strategy.Client.Get(http.DefaultClient, tokenCred, "https://api.twitter.com/1.1/account/verify_credentials.json", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var v twitterResponse
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return "", err
	}

	expectedLink, ok := strategy.Store.Claim(expectedSecret)
	if !ok {
		return "", ErrUnauthorized
	}

	var profileURL string
	for _, entity := range v.Entities.URL.URLs {
		if entity.URL == v.URL {
			profileURL = entity.ExpandedURL
		}
	}

	if !urlsEqual(expectedLink, profileURL) {
		return "", ErrUnauthorized
	}

	return expectedLink, nil
}

// Twitter will respond with something containing, which is stupid but whatever.
//
// {
// 	"url": "https:\/\/t.co\/qsNrcG2afz",
// 	"entities": {
// 		"url": {
// 			"urls":	[
// 				{
// 					"url": "https:\/\/t.co\/qsNrcG2afz",
// 					"expanded_url": "https:\/\/hawx.me",
// 					"display_url": "hawx.me",
// 					"indices": [0,23]
// 				}
// 			]
// 		}
// }
type twitterResponse struct {
	URL      string `json:"url"`
	Entities struct {
		URL struct {
			URLs []struct {
				URL         string `json:"url"`
				ExpandedURL string `json:"expanded_url"`
			} `json:"urls"`
		} `json:"url"`
	} `json:"entities"`
}
