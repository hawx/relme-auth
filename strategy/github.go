package strategy

import (
	"context"
	"encoding/json"
	"net/url"

	"golang.org/x/oauth2"
)

type authGitHub struct {
	conf   *oauth2.Config
	store  strategyStore
	apiURI string
}

// GitHub provides a strategy for authenticating with https://github.com.
func GitHub(store strategyStore, id, secret string) Strategy {
	conf := &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Scopes:       []string{},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}

	return &authGitHub{
		conf:   conf,
		store:  store,
		apiURI: "https://api.github.com",
	}
}

func (authGitHub) Name() string {
	return "github"
}

func (authGitHub) Match(profile *url.URL) bool {
	return profile.Hostname() == "github.com"
}

func (strategy *authGitHub) Redirect(me, profile string) (redirectURL string, err error) {
	state, err := strategy.store.Insert(me)
	if err != nil {
		return "", err
	}

	return strategy.conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (strategy *authGitHub) Callback(form url.Values) (string, error) {
	data, ok := strategy.store.Claim(form.Get("state"))
	if !ok {
		return "", ErrUnknown
	}
	expectedURL := data.(string)

	ctx := context.Background()

	tok, err := strategy.conf.Exchange(ctx, form.Get("code"))
	if err != nil {
		return "", err
	}

	client := strategy.conf.Client(ctx, tok)
	resp, err := client.Get(strategy.apiURI + "/user")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var v gitHubResponse
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return "", err
	}

	if !urlsEqual(v.Blog, expectedURL) {
		return "", ErrUnauthorized
	}

	return expectedURL, nil
}

type gitHubResponse struct {
	Blog string `json:"blog"`
}
