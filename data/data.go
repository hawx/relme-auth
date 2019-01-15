// Package data defines the structures required for storing session state
//
// See subpackages for actual implementations
package data

import "time"

// A Database is everything that should be implemented for subpackages. I know,
// this isn't very Go...
type Database interface {
	SessionStore
	StrategyStore
	CacheStore
	Close() error
}

// A CacheStore allows caching for user and client info speeding up the app when
// requests are made for the same things.
type CacheStore interface {
	CacheProfile(Profile) error
	CacheClient(Client) error
	GetProfile(me string) (Profile, error)
	GetClient(clientID string) (Client, error)
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
	UpdatedAt   time.Time

	Name string
}

// SessionStore is used by relme-auth to keep track of current user sessions
// when initiating authentication or verifying a user's identity.
type SessionStore interface {
	Save(*Session)
	Get(string) (Session, bool)
	GetByCode(string) (Session, bool)
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
	Me          string
	Provider    string
	ProfileURI  string
	ClientID    string
	RedirectURI string
	Code        string
	CreatedAt   time.Time
}

// Expired returns true if the Session was created over 60 seconds ago.
func (s Session) Expired() bool {
	return time.Now().Add(-60 * time.Second).After(s.CreatedAt)
}