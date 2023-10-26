package data

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
	// register sqlite3 for database/sql
	_ "github.com/mattn/go-sqlite3"
)

type Expiry struct {
	// Session specifies how long a session should be valid for. This is the time
	// from the start of authentication (being served the "choose" page), and
	// hitting the callback from an auth provider.
	Session time.Duration

	// Code specifies how long a code should be valid for. This is the time
	// between hitting the callback from an auth provider, and the client
	// verifying the code.
	Code time.Duration

	// Client specifies how long to store information about a client. It has no
	// influence on the authentication session, but outdated information may be
	// misleading.
	Client time.Duration

	// Profile specifies how long to store the authentication methods for a
	// user. This data can be manually refreshed on the "choose" page.
	Profile time.Duration

	// Login specifies how long to consider the user logged in to relme-auth. If a
	// un-expired login is found a user will be presented with the option to
	// "continue" on the "choose" page, bypassing the need to reauthenticate with
	// a downstream provider.
	Login time.Duration
}

type Database struct {
	db         *sql.DB
	httpClient *http.Client
	cookies    sessions.Store
	expiry     Expiry
}

func Open(path string, httpClient *http.Client, cookies sessions.Store, expiry Expiry) (*Database, error) {
	sqlite, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	db := &Database{
		db:         sqlite,
		httpClient: httpClient,
		cookies:    cookies,
		expiry:     expiry,
	}

	return db, db.migrate()
}

func (d *Database) migrate() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS profile (
			Me        TEXT PRIMARY KEY,
			CreatedAt DATETIME
		);

		CREATE TABLE IF NOT EXISTS method (
			Me       TEXT,
			Provider TEXT,
			Profile  TEXT,
			PRIMARY KEY (Me, Provider),
			FOREIGN KEY (Me) REFERENCES profile(Me)
		);

		CREATE TABLE IF NOT EXISTS client (
			ClientID    TEXT PRIMARY KEY,
			RedirectURI TEXT,
			Name        TEXT,
			CreatedAt   DATETIME
		);

		CREATE TABLE IF NOT EXISTS session (
			Me           TEXT PRIMARY KEY,
			ResponseType TEXT,
			Provider     TEXT,
			ProfileURI   TEXT,
			ClientID     TEXT,
			RedirectURI  TEXT,
			Scope        TEXT,
			State        TEXT,
			Code         TEXT,
			CreatedAt    DATETIME
		);

		CREATE TABLE IF NOT EXISTS token (
			ShortToken    TEXT PRIMARY KEY,
			LongTokenHash TEXT,
			Me            TEXT,
			ClientID      TEXT,
			Scope         TEXT,
			CreatedAt     DATETIME
		);

		CREATE TABLE IF NOT EXISTS login (
			ID        TEXT,
			Me        TEXT PRIMARY KEY,
			CreatedAt DATETIME
		);

`)
	if err != nil {
		return err
	}

	version, err := d.schemaVersion()
	if err != nil {
		return err
	}

	stmts := []string{
		`ALTER TABLE session ADD COLUMN CodeChallenge TEXT;
		 ALTER TABLE session ADD COLUMN CodeChallengeMethod TEXT;`,
	}

	for _, stmt := range stmts[version:] {
		_, err := d.db.Exec(stmt)
		if err != nil {
			return err
		}
	}

	return d.setSchemaVersion(len(stmts))
}

func (d *Database) schemaVersion() (int, error) {
	row := d.db.QueryRow("PRAGMA user_version")

	var version int
	err := row.Scan(&version)
	return version, err
}

func (d *Database) setSchemaVersion(version int) error {
	_, err := d.db.Exec("PRAGMA user_version = " + strconv.Itoa(version))
	return err
}

func (d *Database) Forget(me string) error {
	_, err := d.db.Exec(`
		DELETE FROM profile WHERE Me = ?;
		DELETE FROM method WHERE Me = ?;
		DELETE FROM session WHERE Me = ?;
		DELETE FROM token WHERE Me = ?;
		DELETE FROM login WHERE Me = ?;
	`,
		me, me, me, me, me)

	return err
}

func (d *Database) Close() error {
	return d.db.Close()
}
