package data

import (
	"time"
)

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

func (d *Database) CreateSession(session Session) error {
	_, err := d.db.Exec(`
    INSERT OR REPLACE INTO session(ResponseType, Me, ClientID, RedirectURI, Scope, State, Provider, ProfileURI, CreatedAt)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
  `,
		session.ResponseType,
		session.Me,
		session.ClientID,
		session.RedirectURI,
		session.Scope,
		session.State,
		"",
		"",
		session.CreatedAt)

	return err
}

func (d *Database) SetProvider(me, provider, profileURI string) error {
	_, err := d.db.Exec(`UPDATE session SET Provider = ?, ProfileURI = ? WHERE Me = ?`,
		provider,
		profileURI,
		me)

	return err
}

func (d *Database) Session(me string) (session Session, err error) {
	row := d.db.QueryRow(`
    SELECT ResponseType, Me, ClientID, RedirectURI, Scope, State, Provider, ProfileURI, CreatedAt
    FROM session
    WHERE Me = ?`,
		me)

	err = row.Scan(
		&session.ResponseType,
		&session.Me,
		&session.ClientID,
		&session.RedirectURI,
		&session.Scope,
		&session.State,
		&session.Provider,
		&session.ProfileURI,
		&session.CreatedAt)
	return
}