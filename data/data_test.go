package data_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/data/boltdb"
	"hawx.me/code/relme-auth/data/memory"
)

func TestCacheStore(t *testing.T) {
	tmpfile, _ := ioutil.TempFile("", "boltdb")
	defer os.Remove(tmpfile.Name())

	db, _ := boltdb.Open(tmpfile.Name())

	stores := map[string]data.CacheStore{
		"memory": memory.New(),
		"boltdb": db,
	}

	profile := data.Profile{Me: "me", UpdatedAt: time.Now().UTC()}
	client := data.Client{ID: "client", UpdatedAt: time.Now().UTC()}

	for name, store := range stores {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			assert.Nil(store.CacheProfile(profile))

			foundProfile, err := store.GetProfile(profile.Me)
			assert.Nil(err)
			assert.Equal(profile.Me, foundProfile.Me)
			assert.Equal(profile.UpdatedAt, foundProfile.UpdatedAt)

			assert.Nil(store.CacheClient(client))

			foundClient, err := store.GetClient(client.ID)
			assert.Nil(err)
			assert.Equal(client.ID, foundClient.ID)
			assert.Equal(client.UpdatedAt, foundClient.UpdatedAt)
		})
	}
}
