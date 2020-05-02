package data

import (
	"net/http"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestSession(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{})
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
	assert(err).Must.Nil()

	session, err := db.Session("http://john.doe.example.com")
	assert(err).Nil()
	assert(session.ResponseType).Equal("code")
	assert(session.Me).Equal("http://john.doe.example.com")
	assert(session.ClientID).Equal("http://client.example.com")
	assert(session.RedirectURI).Equal("http://client.example.com/callback")
	assert(session.Scope).Equal("create update")
	assert(session.State).Equal("abcde")
	assert(session.CreatedAt).Equal(now)

	err = db.SetProvider("http://john.doe.example.com", "someone", "http://someone.example.com/john.doe")
	assert(err).Must.Nil()

	session, err = db.Session("http://john.doe.example.com")
	assert(err).Must.Nil()
	assert(session.ResponseType).Equal("code")
	assert(session.Me).Equal("http://john.doe.example.com")
	assert(session.ClientID).Equal("http://client.example.com")
	assert(session.RedirectURI).Equal("http://client.example.com/callback")
	assert(session.Scope).Equal("create update")
	assert(session.State).Equal("abcde")
	assert(session.Provider).Equal("someone")
	assert(session.ProfileURI).Equal("http://someone.example.com/john.doe")
	assert(session.CreatedAt).Equal(now)
}
