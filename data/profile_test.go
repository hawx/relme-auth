package data

import (
	"net/http"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestProfile(t *testing.T) {
	assert := assert.New(t)

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
	assert.Nil(err)

	profile, err := db.Profile("http://john.doe.example.com")
	assert.Nil(err)

	assert.Equal("http://john.doe.example.com", profile.Me)
	assert.WithinDuration(now, profile.UpdatedAt, 10*time.Millisecond)
	assert.False(profile.Expired())

	if assert.Len(profile.Methods, 3) {
		assert.Equal("else", profile.Methods[0].Provider)
		assert.Equal("http://else.example.com/john.doe", profile.Methods[0].Profile)
		assert.Equal("other", profile.Methods[1].Provider)
		assert.Equal("http://other.example.com/john.doe", profile.Methods[1].Profile)
		assert.Equal("someone", profile.Methods[2].Provider)
		assert.Equal("http://someone.example.com/john.doe", profile.Methods[2].Profile)
	}

	err = db.CacheProfile(Profile{
		Me:        "http://john.doe.example.com",
		UpdatedAt: now,
		Methods: []Method{
			{Provider: "someone", Profile: "http://someone.example.com/john.doe"},
			{Provider: "other", Profile: "http://cool.example.com/john.doe"},
		},
	})
	assert.Nil(err)

	profile, err = db.Profile("http://john.doe.example.com")
	assert.Nil(err)

	assert.Equal("http://john.doe.example.com", profile.Me)
	assert.WithinDuration(now, profile.UpdatedAt, 10*time.Millisecond)
	assert.False(profile.Expired())

	if assert.Len(profile.Methods, 2) {
		assert.Equal("other", profile.Methods[0].Provider)
		assert.Equal("http://cool.example.com/john.doe", profile.Methods[0].Profile)
		assert.Equal("someone", profile.Methods[1].Provider)
		assert.Equal("http://someone.example.com/john.doe", profile.Methods[1].Profile)
	}
}

func TestProfileWhenExpired(t *testing.T) {
	assert := assert.New(t)

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
	assert.Nil(err)

	profile, err := db.Profile("http://john.doe.example.com")
	assert.Nil(err)

	assert.Equal("http://john.doe.example.com", profile.Me)
	assert.WithinDuration(now, profile.UpdatedAt, 10*time.Millisecond)
	assert.True(profile.Expired())

	if assert.Len(profile.Methods, 3) {
		assert.Equal("else", profile.Methods[0].Provider)
		assert.Equal("http://else.example.com/john.doe", profile.Methods[0].Profile)
		assert.Equal("other", profile.Methods[1].Provider)
		assert.Equal("http://other.example.com/john.doe", profile.Methods[1].Profile)
		assert.Equal("someone", profile.Methods[2].Provider)
		assert.Equal("http://someone.example.com/john.doe", profile.Methods[2].Profile)
	}
}
