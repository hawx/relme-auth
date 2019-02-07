package data

import (
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestProfile(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared")
	defer db.Close()

	now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err := db.CacheProfile(Profile{
		Me:        "http://john.doe.example.com",
		UpdatedAt: now,
		Methods: []Method{
			{Provider: "someone", Profile: "http://someone.example.com/john.doe"},
			{Provider: "else", Profile: "http://else.example.com/john.doe"},
		},
	})
	assert.Nil(err)

	profile, err := db.Profile("http://john.doe.example.com")
	assert.Nil(err)

	assert.Equal("http://john.doe.example.com", profile.Me)
	assert.Equal(now, profile.UpdatedAt)

	if assert.Len(profile.Methods, 2) {
		assert.Equal("else", profile.Methods[0].Provider)
		assert.Equal("http://else.example.com/john.doe", profile.Methods[0].Profile)

		assert.Equal("someone", profile.Methods[1].Provider)
		assert.Equal("http://someone.example.com/john.doe", profile.Methods[1].Profile)
	}
}
