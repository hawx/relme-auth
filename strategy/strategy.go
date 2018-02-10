package strategy

import "net/url"

type Strategy interface {
	// Match determines from the found rel="me" links whether this Strategy can be
	// used.
	Match(me *url.URL) bool

	// Redirect returns the URL to redirect the user to begin the authentication flow.
	Redirect() string

	// Callback handles the user's return from the 3rd party auth provider. It
	// returns the profile URL for the authenticated user, hopefully matching the
	// rel="me" link earlier. If it does not match then the user who authenticated
	// with the OAuth provider is different to the user attempting to authenticate
	// with relme-auth.
	Callback(code string) (string, error)
}
