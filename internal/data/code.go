package data

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

var ErrUnknownCodeChallengeMethod = errors.New("code_challenge_method is not understood")

type Code struct {
	Code                string
	ResponseType        string
	Me                  string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	CreatedAt           time.Time
	ExpiresAt           time.Time
}

// Expired returns true if the Code was created over 60 seconds ago.
func (c Code) Expired() bool {
	return time.Now().After(c.ExpiresAt)
}

func base64URLEncode(src []byte) string {
	s := base64.URLEncoding.EncodeToString(src)

	return strings.TrimRight(s, "=")
}

func (c Code) VerifyChallenge(verifier string) (bool, error) {
	switch c.CodeChallengeMethod {
	case "plain":
		return c.CodeChallenge == verifier, nil
	case "S256":
		data := sha256.Sum256([]byte(verifier))

		return base64URLEncode(data[:]) == c.CodeChallenge, nil
	default:
		return false, ErrUnknownCodeChallengeMethod
	}
}

func (d *Database) CreateCode(me, code string, createdAt time.Time) error {
	_, err := d.db.Exec(`UPDATE session SET Code = ?, CreatedAt = ? WHERE Me = ?`,
		code,
		createdAt,
		me)

	return err
}

func (d *Database) Code(c string) (code Code, err error) {
	row := d.db.QueryRow(`SELECT Code, ResponseType, Me, ClientID, RedirectURI, CodeChallenge, CodeChallengeMethod, Scope, CreatedAt FROM session WHERE Code = ?`,
		c)

	err = row.Scan(
		&code.Code,
		&code.ResponseType,
		&code.Me,
		&code.ClientID,
		&code.RedirectURI,
		&code.CodeChallenge,
		&code.CodeChallengeMethod,
		&code.Scope,
		&code.CreatedAt)
	if err != nil {
		return
	}

	code.ExpiresAt = code.CreatedAt.Add(d.expiry.Code)

	_, err = d.db.Exec(`DELETE FROM session WHERE Code = ?`, c)
	return
}
