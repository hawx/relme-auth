package data

import "time"

type Token struct {
	Token     string
	Me        string
	ClientID  string
	Scope     string
	CreatedAt time.Time
}

func (d *Database) CreateToken(token Token) error {
	_, err := d.db.Exec(`INSERT INTO token(Token, Me, ClientID, Scope, CreatedAt) VALUES (?, ?, ?, ?, ?)`,
		token.Token,
		token.Me,
		token.ClientID,
		token.Scope,
		token.CreatedAt)

	return err
}

func (d *Database) Token(t string) (token Token, err error) {
	row := d.db.QueryRow(`SELECT Token, Me, ClientID, Scope, CreatedAt FROM token WHERE Token = ?`,
		t)

	err = row.Scan(
		&token.Token,
		&token.Me,
		&token.ClientID,
		&token.Scope,
		&token.CreatedAt)
	return
}

func (d *Database) RevokeToken(token string) error {
	_, err := d.db.Exec(`DELETE FROM token WHERE Token = ?`, token)

	return err
}
