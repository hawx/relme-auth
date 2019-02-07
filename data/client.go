package data

import (
	"time"
)

// Client stores an app's information, so it doesn't have to be queried again. If
// redirectURI no longer matches then the data is invalidated.
type Client struct {
	ID          string
	RedirectURI string
	Name        string
	UpdatedAt   time.Time
}

func (d *Database) CacheClient(client Client) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO client(ClientID, RedirectURI, Name, CreatedAt) VALUES(?, ?, ?, ?)`,
		client.ID,
		client.RedirectURI,
		client.Name,
		client.UpdatedAt)

	return err
}

func (d *Database) Client(clientID string) (client Client, err error) {
	row := d.db.QueryRow(`SELECT ClientID, RedirectURI, Name, CreatedAt FROM client WHERE ClientID = ?`,
		clientID)

	err = row.Scan(&client.ID, &client.RedirectURI, &client.Name, &client.UpdatedAt)
	return
}
