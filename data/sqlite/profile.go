package sqlite

import "hawx.me/code/relme-auth/data"

func (d *Database) CacheProfile(profile data.Profile) error {
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

func (d *Database) Profile(me string) (data.Profile, error) {
	rows, err := d.db.Query(`
    SELECT profile.Me, profile.CreatedAt, method.Provider, method.Profile
    FROM method
    LEFT JOIN profile ON method.Me = profile.Me
    WHERE method.Me = ?
    ORDER BY method.Provider`, me)
	if err != nil {
		return data.Profile{}, err
	}
	defer rows.Close()

	var profile data.Profile
	for rows.Next() {
		var method data.Method
		if err = rows.Scan(&profile.Me, &profile.UpdatedAt, &method.Provider, &method.Profile); err != nil {
			return profile, err
		}
		profile.Methods = append(profile.Methods, method)
	}

	return profile, rows.Err()
}
