package strategy

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/garyburd/go-oauth/oauth"
	"hawx.me/code/relme-auth/data"
)

type authFlickr struct {
	Client      oauth.Client
	ApiKey      string
	CallbackURL string
	Store       data.StrategyStore
	ApiURI      string
}

func Flickr(baseURL string, store data.StrategyStore, id, secret string) Strategy {
	oauthClient := oauth.Client{
		TemporaryCredentialRequestURI: "https://www.flickr.com/services/oauth/request_token",
		ResourceOwnerAuthorizationURI: "https://www.flickr.com/services/oauth/authorize",
		TokenRequestURI:               "https://www.flickr.com/services/oauth/access_token",
		Credentials: oauth.Credentials{
			Token:  id,
			Secret: secret,
		},
	}

	return &authFlickr{
		Client:      oauthClient,
		CallbackURL: baseURL + "/oauth/callback/flickr",
		Store:       store,
		ApiKey:      id,
		ApiURI:      "https://api.flickr.com/services/rest",
	}
}

func (authFlickr) Name() string {
	return "flickr"
}

func (authFlickr) Match(me *url.URL) bool {
	return me.Hostname() == "www.flickr.com"
}

func (strategy *authFlickr) Redirect(expectedURL string) (redirectURL string, err error) {
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

	return strategy.Client.AuthorizationURL(tempCred, url.Values{"perms": {"read"}}), nil
}

func (strategy *authFlickr) Callback(form url.Values) (string, error) {
	oauthToken := form.Get("oauth_token")
	expectedSecret, ok := strategy.Store.Claim(oauthToken)
	if !ok {
		return "", errors.New("Unknown oauth_token")
	}

	tempCred := &oauth.Credentials{
		Token:  oauthToken,
		Secret: expectedSecret,
	}
	tokenCred, vals, err := strategy.Client.RequestToken(http.DefaultClient, tempCred, form.Get("oauth_verifier"))
	if err != nil {
		return "", errors.New("Error getting request token, " + err.Error())
	}

	nsid := vals.Get("user_nsid")

	resp, err := strategy.Client.Get(http.DefaultClient, tokenCred, strategy.ApiURI, url.Values{
		"nojsoncallback": {"1"},
		"format":         {"json"},
		"api_key":        {strategy.ApiKey},
		"user_id":        {nsid},
		"method":         {"flickr.profile.getProfile"},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var v flickrResponse
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return "", err
	}

	expectedLink, ok := strategy.Store.Claim(expectedSecret)
	if !ok || !urlsEqual(expectedLink, v.Profile.Website) {
		return "", ErrUnauthorized
	}

	return expectedLink, nil
}

type flickrResponse struct {
	Profile struct {
		Website string `json:"website"`
	} `json:"profile"`
}
