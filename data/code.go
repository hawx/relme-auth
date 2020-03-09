package data

import (
	"time"
)

type Code struct {
	Code         string
	ResponseType string
	Me           string
	ClientID     string
	RedirectURI  string
	Scope        string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// Expired returns true if the Code was created over 60 seconds ago.
func (c Code) Expired() bool {
	return time.Now().After(c.ExpiresAt)
}

func (d *Database) CreateCode(me, code string, createdAt time.Time) error {
	_, err := d.db.Exec(`UPDATE session SET Code = ?, CreatedAt = ? WHERE Me = ?`,
		code,
		createdAt,
		me)

	return err
}

func (d *Database) Code(c string) (code Code, err error) {
	row := d.db.QueryRow(`SELECT Code, ResponseType, Me, ClientID, RedirectURI, Scope, CreatedAt FROM session WHERE Code = ?`,
		c)

	err = row.Scan(
		&code.Code,
		&code.ResponseType,
		&code.Me,
		&code.ClientID,
		&code.RedirectURI,
		&code.Scope,
		&code.CreatedAt)
	if err != nil {
		return
	}

	code.ExpiresAt = code.CreatedAt.Add(d.expiry.Code)

	_, err = d.db.Exec(`DELETE FROM session WHERE Code = ?`, c)
	return
}
