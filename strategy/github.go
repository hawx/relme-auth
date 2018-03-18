package strategy

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"hawx.me/code/relme-auth/state"

	"golang.org/x/oauth2"
)

type authGitHub struct {
	Conf   *oauth2.Config
	Store  state.StrategyStore
	ApiURI string
}

func GitHub(store state.StrategyStore, id, secret string) Strategy {
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
		Conf:   conf,
		Store:  store,
		ApiURI: "https://api.github.com",
	}
}

func (authGitHub) Name() string {
	return "github"
}

func (authGitHub) Match(me *url.URL) bool {
	return me.Hostname() == "github.com"
}

func (strategy *authGitHub) Redirect(expectedLink string) (redirectURL string, err error) {
	state, err := strategy.Store.Insert(expectedLink)
	if err != nil {
		return "", err
	}

	return strategy.Conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (strategy *authGitHub) Callback(form url.Values) (string, error) {
	expectedURL, ok := strategy.Store.Claim(form.Get("state"))
	if !ok {
		return "", errors.New("How did you get here?")
	}

	ctx := context.Background()

	tok, err := strategy.Conf.Exchange(ctx, form.Get("code"))
	if err != nil {
		return "", err
	}

	client := strategy.Conf.Client(ctx, tok)
	resp, err := client.Get(strategy.ApiURI + "/user")
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
