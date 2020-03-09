package data

import (
	"errors"
	"net/http"
	"time"

	"hawx.me/code/relme-auth/random"
)

var ErrNoLogin = errors.New("no login exists")

type Login struct {
	ID        string
	Me        string
	CreatedAt time.Time
	expiresAt time.Time
}

func (l Login) Expired() bool {
	return time.Now().After(l.expiresAt)
}

// Login returns a user's profile URL (i.e. 'me' value), if they have recently
// logged in with relme-auth.
func (d *Database) Login(r *http.Request) (string, error) {
	cookie, _ := d.cookies.Get(r, "relme-auth")

	id, ok := cookie.Values["login_id"]
	if !ok {
		return "", ErrNoLogin
	}

	loginID, ok := id.(string)
	if !ok {
		return "", ErrNoLogin
	}

	var login Login

	row := d.db.QueryRow(`SELECT ID, Me, CreatedAt FROM login WHERE ID = ?`, loginID)
	if err := row.Scan(&login.ID, &login.Me, &login.CreatedAt); err != nil {
		return "", ErrNoLogin
	}
	login.expiresAt = login.CreatedAt.Add(d.expiry.Login)

	if login.Expired() {
		return "", ErrNoLogin
	}

	return login.Me, nil
}

func (d *Database) SaveLogin(w http.ResponseWriter, r *http.Request, me string) error {
	loginID, err := random.String(20)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`INSERT OR REPLACE INTO login(ID, Me, CreatedAt) VALUES (?, ?, ?)`,
		loginID,
		me,
		time.Now().UTC())
	if err != nil {
		return err
	}

	cookie, _ := d.cookies.Get(r, "relme-auth")
	cookie.Values["login_id"] = loginID
	return cookie.Save(r, w)
}
