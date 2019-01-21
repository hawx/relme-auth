package strategy

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrUnauthorized = errors.New("You are not the user you told me you were")
)

type Strategies []Strategy

type Strategy interface {
	// Name returns a unique lowercase alpha string naming the Strategy. This will
	// be passed around as the "provider" parameter.
	Name() string

	// Match determines from the found rel="me" links whether this Strategy can be
	// used.
	Match(me *url.URL) bool

	// Redirect returns the URL to redirect the user to begin the authentication flow.
	Redirect(expectedLink string) (redirectURL string, err error)

	// Callback handles the user's return from the 3rd party auth provider. It
	// returns the profile URL for the authenticated user, hopefully matching the
	// rel="me" link earlier. If it does not match then the user who authenticated
	// with the OAuth provider is different to the user attempting to authenticate
	// with relme-auth.
	Callback(form url.Values) (string, error)
}

// Find iterates through the list of verifiedLinks to profiles checking if there
// is a strategy that can be used to authenticate against it, the first strategy
// that is matched is returned.
func (strategies Strategies) Find(verifiedLinks []string) (found Strategy, expectedLink string, ok bool) {
	for _, link := range verifiedLinks {
		fmt.Printf("me=%s\n", link)
		linkURL, _ := url.Parse(link)

		for _, strategy := range strategies {
			if strategy.Match(linkURL) {
				fmt.Printf("Can authenticate with %s\n", link)
				return strategy, link, true
			}
		}
	}

	return
}

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

// Allowed checks every profile in verifiedLinks against the strategies
// returning a map of profile URL to strategy that can be used to authenticate
// against it.
func (strategies Strategies) Allowed(verifiedLinks []string) (found map[string]Strategy, any bool) {
	found = map[string]Strategy{}

	for _, link := range verifiedLinks {
		linkURL, err := url.Parse(link)
		if err != nil {
			continue
		}

		for _, strategy := range strategies {
			if strategy.Match(linkURL) {
				found[link] = strategy
				any = true
			}
		}
	}

	return found, any
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
