package data

import "time"

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

func (d *Database) CreateCode(code Code) error {
	_, err := d.db.Exec(`INSERT INTO code(Code, ResponseType, Me, ClientID, RedirectURI, Scope, CreatedAt) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		code.Code,
		code.ResponseType,
		code.Me,
		code.ClientID,
		code.RedirectURI,
		code.Scope,
		code.CreatedAt)

	return err
}

func (d *Database) Code(c string) (code Code, err error) {
	row := d.db.QueryRow(`SELECT Code, ResponseType, Me, ClientID, RedirectURI, Scope, CreatedAt FROM code WHERE Code = ?`,
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
