package strategy

import (
	"golang.org/x/oauth2"
	"net/url"
	"context"
	"encoding/json"
)

type authGitHub struct {
	Conf *oauth2.Config
}

func GitHub(id, secret string) Strategy {
	conf := &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Scopes:       []string{},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}

	return &authGitHub{Conf: conf}
}

func (strategy *authGitHub) Match(me *url.URL) bool {
	return me.Hostname() == "github.com"
}

func (strategy *authGitHub) Redirect(state string) string {
	return strategy.Conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (strategy *authGitHub) Callback(code string) (string, error) {
	ctx := context.Background()

	tok, err := strategy.Conf.Exchange(ctx, code)
	if err != nil {
		return "", err
	}

	client := strategy.Conf.Client(ctx, tok)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var v userResource
	err = json.NewDecoder(resp.Body).Decode(&v)
	if err != nil {
		return "", err
	}

	return v.URL, nil
}

type userResource struct {
	URL string `json:"html_url"`
}
