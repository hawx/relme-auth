package store

import "time"

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
