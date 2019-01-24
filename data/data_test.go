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

func TestSessionStore(t *testing.T) {
	tmpfile, _ := ioutil.TempFile("", "boltdb")
	defer os.Remove(tmpfile.Name())

	db, _ := boltdb.Open(tmpfile.Name())

	stores := map[string]data.SessionStore{
		"memory": memory.New(),
		"boltdb": db,
	}

	for name, store := range stores {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			originalSession := data.Session{
				Me:          "me",
				RedirectURI: "https://example.com/callback",
			}
			store.Save(&originalSession)

			_, ok := store.Get("fake")
			assert.False(ok)

			gotSession, ok := store.Get("me")
			assert.True(ok)
			assert.Equal("https://example.com/callback", gotSession.RedirectURI)

			gotSession.Provider = "service"
			gotSession.Code = "1234"
			store.Update(gotSession)

			gotAgainSession, ok := store.Get("me")
			assert.True(ok)
			assert.Equal("https://example.com/callback", gotAgainSession.RedirectURI)
			assert.Equal("service", gotAgainSession.Provider)

			_, ok = store.GetByCode("999")
			assert.False(ok)

			gotByCodeSession, ok := store.GetByCode("1234")
			assert.True(ok)
			assert.Equal("https://example.com/callback", gotByCodeSession.RedirectURI)
			assert.Equal("service", gotByCodeSession.Provider)
		})
	}
}

func TestStrategyStore(t *testing.T) {
	tmpfile, _ := ioutil.TempFile("", "boltdb")
	defer os.Remove(tmpfile.Name())

	db, _ := boltdb.Open(tmpfile.Name())

	stores := map[string]data.StrategyStore{
		"memory": memory.New(),
		"boltdb": db,
	}

	for name, store := range stores {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			_, ok := store.Claim("something")
			assert.False(ok)

			state, err := store.Insert("http://example.com")
			assert.Nil(err)

			link, ok := store.Claim(state)
			assert.True(ok)
			assert.Equal("http://example.com", link)

			_, ok = store.Claim(state)
			assert.False(ok)

			assert.Nil(store.Set("keys", "values"))
			link, ok = store.Claim("keys")
			assert.True(ok)
			assert.Equal("values", link)

			_, ok = store.Claim("keys")
			assert.False(ok)
		})
	}
}
