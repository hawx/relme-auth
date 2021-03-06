package data

import (
	"errors"
	"net/url"
	"time"

	"github.com/peterhellberg/link"
	"hawx.me/code/relme-auth/internal/microformats"
)

// Client stores an app's information, so it doesn't have to be queried again. If
// redirectURI no longer matches then the data is invalidated.
type Client struct {
	ID          string
	RedirectURI string
	Name        string
	UpdatedAt   time.Time
	expiresAt   time.Time
}

func (c Client) Expired() bool {
	return time.Now().After(c.expiresAt)
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
	client.expiresAt = client.UpdatedAt.Add(d.expiry.Client)

	return
}

func (d *Database) queryClient(clientID, redirectURI string) (client Client, err error) {
	parsedClientID, err := url.Parse(clientID)
	if err != nil {
		return
	}

	parsedRedirectURI, err := url.Parse(redirectURI)
	if err != nil {
		return
	}

	redirectOK := parsedClientID.Scheme == parsedRedirectURI.Scheme && parsedClientID.Host == parsedRedirectURI.Host

	now := time.Now().UTC()

	if redirectOK && parsedClientID.Hostname() == "localhost" {
		return Client{
			ID:          clientID,
			RedirectURI: redirectURI,
			Name:        "Local App",
			UpdatedAt:   now,
			expiresAt:   now.Add(d.expiry.Client),
		}, nil
	}

	client = Client{
		ID:          clientID,
		RedirectURI: redirectURI,
		Name:        clientID,
		UpdatedAt:   now,
		expiresAt:   now.Add(d.expiry.Client),
	}

	clientInfoResp, err := d.httpClient.Get(clientID)
	if err != nil {
		return
	}
	defer clientInfoResp.Body.Close()

	app, okerr := microformats.ParseApp(clientInfoResp.Body, parsedClientID)
	if okerr == nil {
		client.Name = app.Name
	}

	if !redirectOK {
		whitelist := app.RedirectURIs

		if whitelistedRedirect, ok := link.ParseResponse(clientInfoResp)["redirect_uri"]; ok {
			whitelist = append(whitelist, whitelistedRedirect.URI)
		}

		for _, candidate := range whitelist {
			if candidate == redirectURI {
				redirectOK = true
				break
			}
		}
	}

	if !redirectOK {
		err = errors.New("bad redirect_uri")
	}

	return
}
