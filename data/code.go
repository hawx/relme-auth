package data

import "time"

const codeExpiry = -60 * time.Second

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
	return time.Now().Add(codeExpiry).After(c.CreatedAt)
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
	return
}
