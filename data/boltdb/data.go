package boltdb

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
	"hawx.me/code/relme-auth/data"
)

const (
	profileBucket = "profiles"
	clientBucket  = "clients"
)

func Open(path string) (data.Database, error) {
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

func (d *database) CacheProfile(profile data.Profile) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(profile)
		if err != nil {
			return err
		}

		b := tx.Bucket([]byte(profileBucket))
		return b.Put([]byte(profile.Me), v)
	})
}

func (d *database) CacheClient(client data.Client) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(client)
		if err != nil {
			return err
		}

		b := tx.Bucket([]byte(clientBucket))
		return b.Put([]byte(client.ID), v)
	})
}

func (d *database) GetProfile(me string) (profile data.Profile, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(profileBucket))
		v := b.Get([]byte(me))
		if len(v) == 0 {
			return errors.New("no such profile")
		}

		return json.Unmarshal(v, &profile)
	})

	return profile, err
}

func (d *database) GetClient(clientID string) (client data.Client, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clientBucket))
		v := b.Get([]byte(clientID))
		if len(v) == 0 {
			return errors.New("no such client")
		}

		return json.Unmarshal(v, &client)
	})

	return client, err
}
