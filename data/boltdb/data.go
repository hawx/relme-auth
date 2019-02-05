package boltdb

import (
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

// Open returns the bolt database at the path specified, or creates a new one.
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
	session.Code, _ = data.RandomString(16)
	session.Token, _ = data.RandomString(32)

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

func (d *database) Update(session data.Session) {
	d.db.Update(func(tx *bolt.Tx) error {
		v, err := json.Marshal(session)
		if err != nil {
			return err
		}

		b := tx.Bucket([]byte(sessionBucket))
		b.Put([]byte(session.Me), v)
		b.Put([]byte(session.Code), v)
		b.Put([]byte(session.Token), v)
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

func (d *database) GetByToken(token string) (session data.Session, ok bool) {
	d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(sessionBucket))
		v := b.Get([]byte(token))
		if len(v) == 0 {
			return nil
		}

		ok = true
		return json.Unmarshal(v, &session)
	})

	return
}

func (d *database) RevokeByToken(token string) {
	d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(sessionBucket))
		v := b.Get([]byte(token))
		if len(v) == 0 {
			return nil
		}

		var session data.Session
		if err := json.Unmarshal(v, &session); err == nil {
			b.Delete([]byte(token))
			b.Delete([]byte(session.Code))
			b.Delete([]byte(session.Me))
		}

		return nil
	})
}

type strategyStore struct {
	db     *bolt.DB
	bucket []byte
}

func (d *database) Strategy(name string) (data.StrategyStore, error) {
	bucket := []byte(stateBucket + "/" + name)

	err := d.db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return fmt.Errorf("create %s bucket: %s", bucket, err)
		}
		return nil
	})

	return &strategyStore{db: d.db, bucket: bucket}, err
}

func (s *strategyStore) Insert(link string) (state string, err error) {
	state, err = data.RandomString(64)
	if err != nil {
		return
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		return b.Put([]byte(state), []byte(link))
	})

	return
}

func (s *strategyStore) Set(key, value string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		return b.Put([]byte(key), []byte(value))
	})
}

func (s *strategyStore) Claim(state string) (link string, ok bool) {
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)

		link = string(b.Get([]byte(state)))
		return b.Delete([]byte(state))
	})

	if err != nil {
		return "", false
	}

	return link, link != ""
}
