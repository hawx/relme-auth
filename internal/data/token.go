package data

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

const (
	tokenPrefix   = "relmeauth"
	shortTokenLen = 8
	longTokenLen  = 24
)

func NewToken(generator func(int) (string, error), code Code) (Token, string, error) {
	shortToken, err := generator(shortTokenLen)
	if err != nil {
		return Token{}, "", err
	}

	longToken, err := generator(longTokenLen)
	if err != nil {
		return Token{}, "", err
	}

	return Token{
		ShortToken:    shortToken,
		LongTokenHash: hashToken(longToken),
		Me:            code.Me,
		ClientID:      code.ClientID,
		Scope:         code.Scope,
		CreatedAt:     time.Now(),
	}, tokenPrefix + "_" + shortToken + "_" + longToken, nil
}

type Token struct {
	ShortToken    string
	LongTokenHash string
	Me            string
	ClientID      string
	Scope         string
	CreatedAt     time.Time
}

func hashToken(t string) string {
	tokenHash := sha256.Sum256([]byte(t))

	return base64.RawStdEncoding.EncodeToString(tokenHash[:])
}

func (d *Database) CreateToken(token Token) error {
	_, err := d.db.Exec(`INSERT INTO token(ShortToken, LongTokenHash, Me, ClientID, Scope, CreatedAt) VALUES (?, ?, ?, ?, ?, ?)`,
		token.ShortToken,
		token.LongTokenHash,
		token.Me,
		token.ClientID,
		token.Scope,
		token.CreatedAt)

	return err
}

func (d *Database) Token(t string) (token Token, err error) {
	parts := strings.Split(t, "_")
	if len(parts) != 3 && parts[0] != tokenPrefix {
		return token, errors.New("invalid token")
	}

	row := d.db.QueryRow(`SELECT ShortToken, LongTokenHash, Me, ClientID, Scope, CreatedAt FROM token WHERE ShortToken = ? AND LongTokenHash = ?`,
		parts[1], hashToken(parts[2]))

	err = row.Scan(
		&token.ShortToken,
		&token.LongTokenHash,
		&token.Me,
		&token.ClientID,
		&token.Scope,
		&token.CreatedAt)
	return
}

func (d *Database) Tokens(me string) (tokens []Token, err error) {
	rows, err := d.db.Query(`SELECT ShortToken, Me, ClientID, Scope, CreatedAt FROM token WHERE Me = ?`,
		me)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var token Token
		if err = rows.Scan(
			&token.ShortToken,
			&token.Me,
			&token.ClientID,
			&token.Scope,
			&token.CreatedAt,
		); err != nil {
			return
		}

		tokens = append(tokens, token)
	}

	err = rows.Err()
	return
}

func (d *Database) RevokeToken(shortToken string) error {
	_, err := d.db.Exec(`DELETE FROM token WHERE ShortToken = ?`, shortToken)

	return err
}

func (d *Database) RevokeClient(me, clientID string) error {
	_, err := d.db.Exec(`DELETE FROM token WHERE Me = ? AND ClientID = ?`, me, clientID)

	return err
}
