// Package data defines the structures required for storing session state
//
// See subpackages for actual implementations
package data

import (
	"crypto/rand"
	"time"
)

// A Database is everything that should be implemented for subpackages. I know,
// this isn't very Go...
type Database interface {
	Strategy(name string) (StrategyStore, error)
	Close() error
}

// Profile stores a user's authentication methods, so they don't have to be
// queried again.
type Profile struct {
	Me        string
	UpdatedAt time.Time

	Methods []Method
}

// Method is a way a user can authenticate, it contains the name of a 3rd party
// provider and the expected profile URL with that provider.
type Method struct {
	Provider string
	Profile  string
}

// Client stores an app's information, so it doesn't have to be queried again. If
// redirectURI no longer matches then the data is invalidated.
type Client struct {
	ID          string
	RedirectURI string
	Name        string
	UpdatedAt   time.Time
}

// StrategyStore is used by strategies to keep track of OAuthy type stuff
// between redirect and callback.
type StrategyStore interface {
	Insert(link string) (state string, err error)
	Set(key, value string) error
	Claim(state string) (link string, ok bool)
}

// Session contains all of the information needed to keep track of OAuth
// requests/responses with a 3rd party.
type Session struct {
	ResponseType string
	Me           string
	Provider     string
	ProfileURI   string
	ClientID     string
	RedirectURI  string
	Scope        string
	State        string
	CreatedAt    time.Time
}

const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"

// RandomString produces a random string of n characters.
func RandomString(n int) (string, error) {
	bytes, err := randomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}

func randomBytes(length int) (b []byte, err error) {
	b = make([]byte, length)
	_, err = rand.Read(b)
	return
}

type Token struct {
	Token     string
	Me        string
	ClientID  string
	Scope     string
	CreatedAt time.Time
}

type Code struct {
	Code         string
	ResponseType string
	Me           string
	ClientID     string
	RedirectURI  string
	Scope        string
	CreatedAt    time.Time
}

// Expired returns true if the Code was created over 60 seconds ago.
func (c Code) Expired() bool {
	return time.Now().Add(-60 * time.Second).After(c.CreatedAt)
}
