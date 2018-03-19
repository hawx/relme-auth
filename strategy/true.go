package strategy

import "net/url"

type authTrue struct{}

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
