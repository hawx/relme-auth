package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
)

const (
	profileBucket = "profiles"
	clientBucket  = "clients"
)

// Profile stores a user's authentication methods, so they don't have to be
// queried again.
type Profile struct {
	Me        string
	UpdatedAt time.Time

	Methods []Method
}

type Method struct {
	Provider string
	Profile  string
}

// Client stores an app's information, so it doesn't have to be queried again. If
// redirectURI no longer matches then the data is invalidated.
type Client struct {
	ID          string
	RedirectURI string
	UpdatedAt   time.Time

	Name string
}

type Database interface {
	CacheProfile(Profile) error
	CacheClient(Client) error
	GetProfile(me string) (Profile, error)
	GetClient(clientID string) (Client, error)
	Close() error
}

func Open(path string) (Database, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(profileBucket)); err != nil {
			return fmt.Errorf("create profile bucket: %s", err)
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(clientBucket)); err != nil {
			return fmt.Errorf("create client bucket: %s", err)
		}

		return nil
	})

	return &database{db: db}, err
}

type database struct{ db *bolt.DB }

func (d *database) Close() error {
	return d.db.Close()
}

func (d *database) CacheProfile(profile Profile) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(profile)
		if err != nil {
			return err
		}

		b := tx.Bucket([]byte(profileBucket))
		return b.Put([]byte(profile.Me), v)
	})
}

func (d *database) CacheClient(client Client) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(client)
		if err != nil {
			return err
		}

		b := tx.Bucket([]byte(clientBucket))
		return b.Put([]byte(client.ID), v)
	})
}

func (d *database) GetProfile(me string) (Profile, error) {
	var profile Profile

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(profileBucket))
		v := b.Get([]byte(me))
		if len(v) == 0 {
			return errors.New("no such profile")
		}

		err := json.Unmarshal(v, &profile)
		return err
	})

	return profile, err
}

func (d *database) GetClient(clientID string) (Client, error) {
	var client Client

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clientBucket))
		v := b.Get([]byte(clientID))
		if len(v) == 0 {
			return errors.New("no such client")
		}

		err := json.Unmarshal(v, &client)
		return err
	})

	return client, err
}
