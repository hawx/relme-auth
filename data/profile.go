package data

import (
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
	_, err := d.db.Exec(`INSERT OR REPLACE INTO profile(Me, CreatedAt) VALUES(?, ?)`,
		profile.Me,
		profile.UpdatedAt)

	for _, method := range profile.Methods {
		_, err = d.db.Exec(`INSERT OR REPLACE INTO method(Me, Provider, Profile) VALUES(?, ?, ?)`,
			profile.Me,
			method.Provider,
			method.Profile)
	}

	return err
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
	for rows.Next() {
		var method Method
		if err = rows.Scan(&profile.Me, &profile.UpdatedAt, &method.Provider, &method.Profile); err != nil {
			return profile, err
		}
		profile.Methods = append(profile.Methods, method)
	}

	return profile, rows.Err()
}
