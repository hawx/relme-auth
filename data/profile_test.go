package data

import (
	"net/http"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestProfile(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{Profile: time.Hour})
	defer db.Close()

	now := time.Now()
	err := db.CacheProfile(Profile{
		Me:        "http://john.doe.example.com",
		UpdatedAt: now,
		Methods: []Method{
			{Provider: "someone", Profile: "http://someone.example.com/john.doe"},
			{Provider: "else", Profile: "http://else.example.com/john.doe"},
			{Provider: "other", Profile: "http://other.example.com/john.doe"},
		},
	})
	assert(err).Nil()

	profile, err := db.Profile("http://john.doe.example.com")
	assert(err).Nil()

	assert(profile.Me).Equal("http://john.doe.example.com")
	assert(profile.UpdatedAt).WithinDuration(now, 10*time.Millisecond)
	assert(profile.Expired()).False()

	if assert(profile.Methods).Len(3) {
		assert(profile.Methods[0].Provider).Equal("else")
		assert(profile.Methods[0].Profile).Equal("http://else.example.com/john.doe")
		assert(profile.Methods[1].Provider).Equal("other")
		assert(profile.Methods[1].Profile).Equal("http://other.example.com/john.doe")
		assert(profile.Methods[2].Provider).Equal("someone")
		assert(profile.Methods[2].Profile).Equal("http://someone.example.com/john.doe")
	}

	err = db.CacheProfile(Profile{
		Me:        "http://john.doe.example.com",
		UpdatedAt: now,
		Methods: []Method{
			{Provider: "someone", Profile: "http://someone.example.com/john.doe"},
			{Provider: "other", Profile: "http://cool.example.com/john.doe"},
		},
	})
	assert(err).Nil()

	profile, err = db.Profile("http://john.doe.example.com")
	assert(err).Nil()

	assert(profile.Me).Equal("http://john.doe.example.com")
	assert(profile.UpdatedAt).WithinDuration(now, 10*time.Millisecond)
	assert(profile.Expired()).False()

	if assert(profile.Methods).Len(2) {
		assert(profile.Methods[0].Provider).Equal("other")
		assert(profile.Methods[0].Profile).Equal("http://cool.example.com/john.doe")
		assert(profile.Methods[1].Provider).Equal("someone")
		assert(profile.Methods[1].Profile).Equal("http://someone.example.com/john.doe")
	}
}

func TestProfileWhenExpired(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{Profile: time.Hour})
	defer db.Close()

	now := time.Now().Add(-time.Hour)
	err := db.CacheProfile(Profile{
		Me:        "http://john.doe.example.com",
		UpdatedAt: now,
		Methods: []Method{
			{Provider: "someone", Profile: "http://someone.example.com/john.doe"},
			{Provider: "else", Profile: "http://else.example.com/john.doe"},
			{Provider: "other", Profile: "http://other.example.com/john.doe"},
		},
	})
	assert(err).Nil()

	profile, err := db.Profile("http://john.doe.example.com")
	assert(err).Nil()

	assert(profile.Me).Equal("http://john.doe.example.com")
	assert(profile.UpdatedAt).WithinDuration(now, 10*time.Millisecond)
	assert(profile.Expired()).True()

	if assert(profile.Methods).Len(3) {
		assert(profile.Methods[0].Provider).Equal("else")
		assert(profile.Methods[0].Profile).Equal("http://else.example.com/john.doe")
		assert(profile.Methods[1].Provider).Equal("other")
		assert(profile.Methods[1].Profile).Equal("http://other.example.com/john.doe")
		assert(profile.Methods[2].Provider).Equal("someone")
		assert(profile.Methods[2].Profile).Equal("http://someone.example.com/john.doe")
	}
}
