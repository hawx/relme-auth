package sqlite

import "hawx.me/code/relme-auth/data"

func (d *Database) CreateCode(code data.Code) error {
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

func (d *Database) Code(c string) (code data.Code, err error) {
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
