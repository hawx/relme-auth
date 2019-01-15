// Package data defines the structures required for storing session state
//
// See subpackages for actual implementations
package data

import "time"

type Database interface {
	CacheProfile(Profile) error
	CacheClient(Client) error
	GetProfile(me string) (Profile, error)
	GetClient(clientID string) (Client, error)
	Close() error
}

// Profile stores a user's authentication methods, so they don't have to be
// queried again.
type Profile struct {
	Me        string
	UpdatedAt time.Time

	Methods []Method
}

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

type Session struct {
	Me          string
	Provider    string
	ProfileURI  string
	ClientID    string
	RedirectURI string
	Code        string
	CreatedAt   time.Time
}

func (s Session) Expired() bool {
	return time.Now().Add(-60 * time.Second).After(s.CreatedAt)
}
