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
	apiKey      string
	apiURI      string
	callbackURL string
	client      oauth.Client
	httpClient  *http.Client
	store       strategyStore
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
		apiKey:      id,
		apiURI:      "https://api.flickr.com/services/rest",
		callbackURL: baseURL + "/callback/flickr",
		client:      oauthClient,
		httpClient:  httpClient,
		store:       store,
	}
}

func (authFlickr) Name() string {
	return "flickr"
}

func (authFlickr) Match(profile *url.URL) bool {
	return profile.Hostname() == "www.flickr.com"
}

func (strategy *authFlickr) Redirect(me, profile string) (redirectURL string, err error) {
	tempCred, err := strategy.client.RequestTemporaryCredentials(strategy.httpClient, strategy.callbackURL, nil)
	if err != nil {
		return "", err
	}

	if err := strategy.store.Set(tempCred.Token, flickrData{
		me:     me,
		secret: tempCred.Secret,
	}); err != nil {
		return "", err
	}

	return strategy.client.AuthorizationURL(tempCred, url.Values{"perms": {"read"}}), nil
}

func (strategy *authFlickr) Callback(form url.Values) (string, error) {
	oauthToken := form.Get("oauth_token")
	data, ok := strategy.store.Claim(oauthToken)
	if !ok {
		return "", errors.New("unknown oauth_token")
	}
	fdata := data.(flickrData)

	tempCred := &oauth.Credentials{
		Token:  oauthToken,
		Secret: fdata.secret,
	}
	tokenCred, vals, err := strategy.client.RequestToken(strategy.httpClient, tempCred, form.Get("oauth_verifier"))
	if err != nil {
		return "", errors.New("error getting request token, " + err.Error())
	}

	nsid := vals.Get("user_nsid")

	resp, err := strategy.client.Get(strategy.httpClient, tokenCred, strategy.apiURI, url.Values{
		"nojsoncallback": {"1"},
		"format":         {"json"},
		"api_key":        {strategy.apiKey},
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
