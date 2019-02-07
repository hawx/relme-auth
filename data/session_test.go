package data

import (
	"net/http"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestSession(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient)
	defer db.Close()

	now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	err := db.CreateSession(Session{
		ResponseType: "code",
		Me:           "http://john.doe.example.com",
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://client.example.com/callback",
		Scope:        "create update",
		State:        "abcde",
		CreatedAt:    now,
	})
	assert.Nil(err)

	session, err := db.Session("http://john.doe.example.com")
	assert.Nil(err)
	assert.Equal("code", session.ResponseType)
	assert.Equal("http://john.doe.example.com", session.Me)
	assert.Equal("http://client.example.com", session.ClientID)
	assert.Equal("http://client.example.com/callback", session.RedirectURI)
	assert.Equal("create update", session.Scope)
	assert.Equal("abcde", session.State)
	assert.Equal(now, session.CreatedAt)

	err = db.SetProvider("http://john.doe.example.com", "someone", "http://someone.example.com/john.doe")
	assert.Nil(err)

	session, err = db.Session("http://john.doe.example.com")
	assert.Nil(err)
	assert.Equal("code", session.ResponseType)
	assert.Equal("http://john.doe.example.com", session.Me)
	assert.Equal("http://client.example.com", session.ClientID)
	assert.Equal("http://client.example.com/callback", session.RedirectURI)
	assert.Equal("create update", session.Scope)
	assert.Equal("abcde", session.State)
	assert.Equal("someone", session.Provider)
	assert.Equal("http://someone.example.com/john.doe", session.ProfileURI)
	assert.Equal(now, session.CreatedAt)
}
