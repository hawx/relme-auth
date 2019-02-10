package data

import (
	"database/sql"
	"time"
)

const profileExpiry = -7 * 24 * time.Hour

// Profile stores a user's authentication methods, so they don't have to be
// queried again.
type Profile struct {
	Me        string
	UpdatedAt time.Time

	Methods []Method
}

func (p Profile) Expired() bool {
	return time.Now().Add(profileExpiry).After(p.UpdatedAt)
}

// Method is a way a user can authenticate, it contains the name of a 3rd party
// provider and the expected profile URL with that provider.
type Method struct {
	Provider string
	Profile  string
}

func (d *Database) CacheProfile(profile Profile) error {
	_, err := d.db.Exec(`
    DELETE FROM method WHERE Me = ?;
    DELETE FROM profile WHERE Me = ?;
    INSERT INTO profile(Me, CreatedAt) VALUES(?, ?);
  `,
		profile.Me,
		profile.Me,
		profile.Me,
		profile.UpdatedAt)

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO method(Me, Provider, Profile) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}

	for _, method := range profile.Methods {
		_, err = stmt.Exec(profile.Me, method.Provider, method.Profile)
	}

	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			return terr
		}
		return err
	}

	return tx.Commit()
}

func (d *Database) Profile(me string) (Profile, error) {
	rows, err := d.db.Query(`
    SELECT profile.Me, profile.CreatedAt, method.Provider, method.Profile
    FROM method
    LEFT JOIN profile ON method.Me = profile.Me
    WHERE method.Me = ?
    ORDER BY method.Provider`, me)
	if err != nil {
		return Profile{}, err
	}
	defer rows.Close()

	var profile Profile
	var ok bool
	for rows.Next() {
		ok = true
		var method Method
		if err = rows.Scan(&profile.Me, &profile.UpdatedAt, &method.Provider, &method.Profile); err != nil {
			return profile, err
		}
		profile.Methods = append(profile.Methods, method)
	}

	if !ok {
		return Profile{}, sql.ErrNoRows
	}

	return profile, rows.Err()
}
