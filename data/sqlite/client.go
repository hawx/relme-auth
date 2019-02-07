package sqlite

import "hawx.me/code/relme-auth/data"

func (d *Database) CacheClient(client data.Client) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO client(ClientID, RedirectURI, Name, CreatedAt) VALUES(?, ?, ?, ?)`,
		client.ID,
		client.RedirectURI,
		client.Name,
		client.UpdatedAt)

	return err
}

func (d *Database) Client(clientID string) (client data.Client, err error) {
	row := d.db.QueryRow(`SELECT ClientID, RedirectURI, Name, CreatedAt FROM client WHERE ClientID = ?`,
		clientID)

	err = row.Scan(&client.ID, &client.RedirectURI, &client.Name, &client.UpdatedAt)
	return
}
