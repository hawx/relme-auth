package data

import (
	"database/sql"
	"net/http"

	// register sqlite3 for sql
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db         *sql.DB
	httpClient *http.Client
}

func Open(path string, httpClient *http.Client) (*Database, error) {
	sqlite, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	db := &Database{db: sqlite, httpClient: httpClient}

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
      Token     TEXT PRIMARY KEY,
      Me        TEXT,
      ClientID  TEXT,
      Scope     TEXT,
      CreatedAt DATETIME
    );
`)

	return err
}

func (d *Database) Close() error {
	return d.db.Close()
}
