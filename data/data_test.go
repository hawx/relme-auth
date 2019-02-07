package data_test

import (
	"testing"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/data/memory"
)

func TestStrategyStore(t *testing.T) {
	memS, _ := memory.New().Strategy("cool")

	stores := map[string]data.StrategyStore{
		"memory": memS,
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
