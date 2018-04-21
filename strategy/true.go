package strategy

import "net/url"

type authTrue struct{}

// True should only be used for testing purposes, it says that everyone is authenticated!
func True() Strategy {
	return authTrue{}
}

func (authTrue) Name() string {
	return "true"
}

func (authTrue) Match(me *url.URL) bool {
	return true
}

func (authTrue) Redirect(expectedURL string) (redirectURL string, err error) {
	redirectURL = "http://localhost:8080/oauth/callback/true?" +
		url.Values{"expected": {expectedURL}}.Encode()

	return redirectURL, nil
}

func (authTrue) Callback(form url.Values) (string, error) {
	return form.Get("expected"), nil
}
