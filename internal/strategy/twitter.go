package strategy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/garyburd/go-oauth/oauth"
)

type twitterData struct {
	me     string
	secret string
}

type authTwitter struct {
	client      oauth.Client
	callbackURL string
	store       strategyStore
	apiURI      string
	httpClient  *http.Client
}

// Twitter provides a strategy for authenticating with https://twitter.com.
func Twitter(baseURL string, store strategyStore, id, secret string, httpClient *http.Client) Strategy {
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
		client:      oauthClient,
		callbackURL: baseURL + "/callback/twitter",
		store:       store,
		apiURI:      "https://api.twitter.com/1.1",
		httpClient:  httpClient,
	}
}

func (authTwitter) Name() string {
	return "twitter"
}

func (authTwitter) Match(profile *url.URL) bool {
	return profile.Hostname() == "twitter.com"
}

func (strategy *authTwitter) Redirect(me, profile string) (redirectURL string, err error) {
	tempCred, err := strategy.client.RequestTemporaryCredentials(strategy.httpClient, strategy.callbackURL, nil)
	if err != nil {
		return "", err
	}

	if err := strategy.store.Set(tempCred.Token, twitterData{
		me:     me,
		secret: tempCred.Secret,
	}); err != nil {
		return "", err
	}

	return strategy.client.AuthorizationURL(tempCred, nil), nil
}

func (strategy *authTwitter) Callback(form url.Values) (string, error) {
	oauthToken := form.Get("oauth_token")
	data, ok := strategy.store.Claim(oauthToken)
	if !ok {
		return "", ErrUnknown
	}
	fdata := data.(twitterData)

	tempCred := &oauth.Credentials{
		Token:  oauthToken,
		Secret: fdata.secret,
	}
	tokenCred, _, err := strategy.client.RequestToken(strategy.httpClient, tempCred, form.Get("oauth_verifier"))
	if err != nil {
		return "", fmt.Errorf("error getting request token: %w", err)
	}

	resp, err := strategy.client.Get(strategy.httpClient, tokenCred, strategy.apiURI+"/account/verify_credentials.json", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var v twitterResponse
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return "", err
	}

	var profileURL string
	for _, entity := range v.Entities.URL.URLs {
		if entity.URL == v.URL {
			profileURL = entity.ExpandedURL
		}
	}

	if !urlsEqual(fdata.me, profileURL) {
		return "", ErrUnauthorized
	}

	return fdata.me, nil
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
