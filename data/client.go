package data

import (
	"time"

	"hawx.me/code/relme-auth/microformats"
)

const clientExpiry = -24 * time.Hour

// Client stores an app's information, so it doesn't have to be queried again. If
// redirectURI no longer matches then the data is invalidated.
type Client struct {
	ID          string
	RedirectURI string
	Name        string
	UpdatedAt   time.Time
}

func (c Client) Expired() bool {
	return time.Now().Add(clientExpiry).After(c.UpdatedAt)
}

func (d *Database) Client(clientID, redirectURI string) (Client, error) {
	client, err := d.findClient(clientID, redirectURI)
	if err != nil || client.Expired() {
		client, err = d.queryClient(clientID, redirectURI)
	}
	if err == nil {
		err = d.cacheClient(client)
	}

	return client, err
}

func (d *Database) cacheClient(client Client) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO client(ClientID, RedirectURI, Name, CreatedAt) VALUES(?, ?, ?, ?)`,
		client.ID,
		client.RedirectURI,
		client.Name,
		client.UpdatedAt)

	return err
}

func (d *Database) findClient(clientID, redirectURI string) (client Client, err error) {
	row := d.db.QueryRow(`SELECT ClientID, RedirectURI, Name, CreatedAt FROM client WHERE ClientID = ? AND RedirectURI = ?`,
		clientID,
		redirectURI)

	err = row.Scan(&client.ID, &client.RedirectURI, &client.Name, &client.UpdatedAt)
	return
}

func (d *Database) queryClient(clientID, redirectURI string) (client Client, err error) {
	client.ID = clientID
	client.Name = clientID
	client.UpdatedAt = time.Now().UTC()
	client.RedirectURI = redirectURI

	clientInfoResp, err := d.httpClient.Get(clientID)
	if err != nil {
		return
	}
	defer clientInfoResp.Body.Close()

	if clientName, _, okerr := microformats.HApp(clientInfoResp.Body); okerr == nil {
		client.Name = clientName
	}

	return
}