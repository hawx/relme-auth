package sqlite

import (
	"hawx.me/code/relme-auth/data"
)

func (d *Database) CreateSession(session data.Session) error {
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

func (d *Database) Session(me string) (session data.Session, err error) {
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
