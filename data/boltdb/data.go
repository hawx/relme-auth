package boltdb

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"hawx.me/code/relme-auth/data"
)

const (
	profileBucket = "profiles"
	clientBucket  = "clients"
	sessionBucket = "sessions"
	stateBucket   = "states"
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

		if _, err := tx.CreateBucketIfNotExists([]byte(sessionBucket)); err != nil {
			return fmt.Errorf("create client bucket: %s", err)
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(stateBucket)); err != nil {
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

func (d *database) Save(session *data.Session) {
	session.CreatedAt = time.Now()
	session.Code, _ = randomString(16)

	d.db.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(session)
		if err != nil {
			return err
		}

		b := tx.Bucket([]byte(sessionBucket))
		b.Put([]byte(session.Me), v)
		b.Put([]byte(session.Code), v)
		return nil
	})
}

func (d *database) Get(me string) (session data.Session, ok bool) {
	d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(sessionBucket))
		v := b.Get([]byte(me))
		if len(v) == 0 {
			return nil
		}

		ok = true
		return json.Unmarshal(v, &session)
	})

	return
}

func (d *database) GetByCode(code string) (session data.Session, ok bool) {
	d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(sessionBucket))
		v := b.Get([]byte(code))
		if len(v) == 0 {
			return nil
		}

		ok = true
		return json.Unmarshal(v, &session)
	})

	return
}

func (d *database) Insert(link string) (state string, err error) {
	state, err = randomString(64)
	if err != nil {
		return
	}

	err = d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stateBucket))
		return b.Put([]byte(state), []byte(link))
	})

	return
}

func (d *database) Set(key, value string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stateBucket))
		return b.Put([]byte(key), []byte(value))
	})
}

func (d *database) Claim(state string) (link string, ok bool) {
	err := d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stateBucket))

		link = string(b.Get([]byte(state)))
		return b.Delete([]byte(state))
	})

	if err != nil {
		return "", false
	}

	return link, true
}

const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"

func randomString(n int) (string, error) {
	bytes, err := randomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}

func randomBytes(length int) (b []byte, err error) {
	b = make([]byte, length)
	_, err = rand.Read(b)
	return
}
