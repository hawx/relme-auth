package strategy

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/garyburd/go-oauth/oauth"
)

type twitterData struct {
	me     string
	secret string
}

type authTwitter struct {
	Client      oauth.Client
	CallbackURL string
	Store       strategyStore
	APIURI      string
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
		Client:      oauthClient,
		CallbackURL: baseURL + "/oauth/callback/twitter",
		Store:       store,
		APIURI:      "https://api.twitter.com/1.1",
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
	tempCred, err := strategy.Client.RequestTemporaryCredentials(strategy.httpClient, strategy.CallbackURL, nil)
	if err != nil {
		return "", err
	}

	if err := strategy.Store.Set(tempCred.Token, twitterData{
		me:     me,
		secret: tempCred.Secret,
	}); err != nil {
		return "", err
	}

	return strategy.Client.AuthorizationURL(tempCred, nil), nil
}

func (strategy *authTwitter) Callback(form url.Values) (string, error) {
	oauthToken := form.Get("oauth_token")
	data, ok := strategy.Store.Claim(oauthToken)
	if !ok {
		return "", errors.New("unknown oauth_token")
	}
	fdata := data.(twitterData)

	tempCred := &oauth.Credentials{
		Token:  oauthToken,
		Secret: fdata.secret,
	}
	tokenCred, _, err := strategy.Client.RequestToken(strategy.httpClient, tempCred, form.Get("oauth_verifier"))
	if err != nil {
		return "", errors.New("error getting request token, " + err.Error())
	}

	resp, err := strategy.Client.Get(strategy.httpClient, tokenCred, strategy.APIURI+"/account/verify_credentials.json", nil)
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
