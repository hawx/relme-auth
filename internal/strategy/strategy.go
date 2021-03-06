package strategy

import (
	"errors"
	"net/url"
	"strings"
)

var (
	// ErrUnauthorized is returned when the user was not authenticated.
	ErrUnauthorized = errors.New("you are not the user you told me you were")

	// ErrUnknown is returned when a user seems to have appeared in the middle of
	// an auth flow.
	ErrUnknown = errors.New("how did you get here?")
)

type strategyStore interface {
	Insert(interface{}) (string, error)
	Set(key string, value interface{}) error
	Claim(string) (interface{}, bool)
}

// Strategies is a list of Strategy.
type Strategies []Strategy

// Strategy is something that can provide authentication for a user.
type Strategy interface {
	// Name returns a unique lowercase alpha string naming the Strategy. This will
	// be passed around as the "provider" parameter.
	Name() string

	// Match determines from the found profile whether this Strategy can be used.
	Match(profile *url.URL) bool

	// Redirect returns the URL to redirect the user to begin the authentication flow.
	Redirect(me, profile string) (redirectURL string, err error)

	// Callback handles the user's return from the 3rd party auth provider. It
	// returns the profile URL for the authenticated user, hopefully matching the
	// rel="me" link earlier. If it does not match then the user who authenticated
	// with the OAuth provider is different to the user attempting to authenticate
	// with relme-auth.
	Callback(form url.Values) (string, error)
}

// IsAllowed checks whether a strategy exists for the profile link that can be
// used to authenticate against it.
func (strategies Strategies) IsAllowed(link string) (found Strategy, ok bool) {
	linkURL, err := url.Parse(link)
	if err != nil {
		return
	}

	for _, strategy := range strategies {
		if strategy.Match(linkURL) {
			return strategy, true
		}
	}

	return
}

func urlsEqual(a, b string) bool {
	return normalizeURL(a) == normalizeURL(b)
}

func normalizeURL(profileURL string) string {
	if !strings.HasSuffix(profileURL, "/") {
		return profileURL + "/"
	}

	return profileURL
}
