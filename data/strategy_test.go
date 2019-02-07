package data

import (
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestStrategy(t *testing.T) {
	store, _ := Strategy("cool")

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
}

func TestStrategyExpiry(t *testing.T) {
	assert := assert.New(t)

	store, _ := Strategy("cool")
	store.expiry = 1

	state, err := store.Insert("http://example.com")
	assert.Nil(err)

	<-time.After(500 * time.Millisecond)
	link, ok := store.Claim(state)
	assert.True(ok)
	assert.Equal("http://example.com", link)

	state, err = store.Insert("http://example.com")
	assert.Nil(err)

	<-time.After(2 * time.Second)
	link, ok = store.Claim(state)
	assert.False(ok)
	assert.Equal("", link)

}
