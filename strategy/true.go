package strategy

import "net/url"

type authTrue struct {
	baseURL string
}

// True should only be used for testing purposes, it says that everyone is authenticated!
func True(baseURL string) Strategy {
	return authTrue{
		baseURL: baseURL,
	}
}

func (authTrue) Name() string {
	return "true"
}

func (authTrue) Match(me *url.URL) bool {
	return true
}

func (t authTrue) Redirect(expectedURL string) (redirectURL string, err error) {
	redirectURL = t.baseURL + "/oauth/callback/true?" +
		url.Values{"expected": {expectedURL}}.Encode()

	return redirectURL, nil
}

func (authTrue) Callback(form url.Values) (string, error) {
	return form.Get("expected"), nil
}
