package strategy

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/garyburd/go-oauth/oauth"
)

type flickrData struct {
	me     string
	secret string
}

type authFlickr struct {
	Client      oauth.Client
	APIKey      string
	CallbackURL string
	Store       strategyStore
	APIURI      string
	httpClient  *http.Client
}

// Flickr provides a strategy for authenticating with https://www.flickr.com.
func Flickr(baseURL string, store strategyStore, id, secret string, httpClient *http.Client) Strategy {
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
		APIKey:      id,
		APIURI:      "https://api.flickr.com/services/rest",
		httpClient:  httpClient,
	}
}

func (authFlickr) Name() string {
	return "flickr"
}

func (authFlickr) Match(profile *url.URL) bool {
	return profile.Hostname() == "www.flickr.com"
}

func (strategy *authFlickr) Redirect(me, profile string) (redirectURL string, err error) {
	tempCred, err := strategy.Client.RequestTemporaryCredentials(strategy.httpClient, strategy.CallbackURL, nil)
	if err != nil {
		return "", err
	}

	if err := strategy.Store.Set(tempCred.Token, flickrData{
		me:     me,
		secret: tempCred.Secret,
	}); err != nil {
		return "", err
	}

	return strategy.Client.AuthorizationURL(tempCred, url.Values{"perms": {"read"}}), nil
}

func (strategy *authFlickr) Callback(form url.Values) (string, error) {
	oauthToken := form.Get("oauth_token")
	data, ok := strategy.Store.Claim(oauthToken)
	if !ok {
		return "", errors.New("unknown oauth_token")
	}
	fdata := data.(flickrData)

	tempCred := &oauth.Credentials{
		Token:  oauthToken,
		Secret: fdata.secret,
	}
	tokenCred, vals, err := strategy.Client.RequestToken(strategy.httpClient, tempCred, form.Get("oauth_verifier"))
	if err != nil {
		return "", errors.New("error getting request token, " + err.Error())
	}

	nsid := vals.Get("user_nsid")

	resp, err := strategy.Client.Get(strategy.httpClient, tokenCred, strategy.APIURI, url.Values{
		"nojsoncallback": {"1"},
		"format":         {"json"},
		"api_key":        {strategy.APIKey},
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

	if !ok || !urlsEqual(fdata.me, v.Profile.Website) {
		return "", ErrUnauthorized
	}

	return fdata.me, nil
}

type flickrResponse struct {
	Profile struct {
		Website string `json:"website"`
	} `json:"profile"`
}
