package data

import (
	"fmt"
	"time"
)

type Credential struct {
	Me           string
	CredentialID string
	PublicKey    string
	CreatedAt    time.Time
}

func (d *Database) CreateCredential(me, credentialID, publicKey string, createdAt time.Time) error {
	const method = "data.Database.CreateCredential"

	if _, err := d.db.Exec(`INSERT INTO credential(Me, CredentialID, PublicKey, CreatedAt) VALUES (?, ?, ?, ?)`,
		me,
		credentialID,
		publicKey,
		createdAt,
	); err != nil {
		return fmt.Errorf("%s insert: %v", method, err)
	}

	return nil
}

func (d *Database) Credentials(me string) (credentials []Credential, err error) {
	const method = "data.Database.Credentials"

	rows, err := d.db.Query(`SELECT Me, CredentialID, PublicKey, CreatedAt FROM credential WHERE Me = ?`,
		me)
	if err != nil {
		return credentials, fmt.Errorf("%s select: %v", method, err)
	}
	defer rows.Close()

	for rows.Next() {
		var credential Credential
		if err := rows.Scan(
			&credential.Me,
			&credential.CredentialID,
			&credential.PublicKey,
			&credential.CreatedAt); err != nil {
			return credentials, fmt.Errorf("%s scan: %v", method, err)
		}
		credentials = append(credentials, credential)
	}

	return
}

func (d *Database) DeleteCredential(credentialID string) error {
	const method = "data.Database.DeleteCredential"

	if _, err := d.db.Exec(`DELETE FROM credential WHERE CredentialID = ?`,
		credentialID,
	); err != nil {
		return fmt.Errorf("%s delete: %v", method, err)
	}

	return nil
}
