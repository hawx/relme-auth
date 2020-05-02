package data

import (
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestStrategy(t *testing.T) {
	store, _ := Strategy("cool")

	assert := assert.Wrap(t)

	_, ok := store.Claim("something")
	assert(ok).False()

	state, err := store.Insert("http://example.com")
	assert(err).Nil()

	link, ok := store.Claim(state)
	assert(ok).True()
	assert(link).Equal("http://example.com")

	_, ok = store.Claim(state)
	assert(ok).False()

	assert(store.Set("keys", "values")).Nil()
	link, ok = store.Claim("keys")
	assert(ok).True()
	assert(link).Equal("values")

	_, ok = store.Claim("keys")
	assert(ok).False()
}

func TestStrategyExpiry(t *testing.T) {
	assert := assert.Wrap(t)

	store, _ := Strategy("cool")
	store.expiry = 1

	state, err := store.Insert("http://example.com")
	assert(err).Nil()

	time.Sleep(500 * time.Millisecond)
	link, ok := store.Claim(state)
	assert(ok).True()
	assert(link).Equal("http://example.com")

	state, err = store.Insert("http://example.com")
	assert(err).Nil()

	time.Sleep(2 * time.Second)
	link, ok = store.Claim(state)
	assert(ok).False()
	assert(link).Equal("")
}
