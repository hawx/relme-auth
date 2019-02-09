package data

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestCode(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient)
	defer db.Close()

	now := time.Now()

	err := db.CreateSession(Session{
		ResponseType: "code",
		Me:           "http://john.doe.example.com",
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://client.example.com/callback",
		CreatedAt:    now,
	})
	assert.Nil(err)

	err = db.CreateCode("http://john.doe.example.com", "abcde", now)
	assert.Nil(err)

	code, err := db.Code("abcde")
	assert.Nil(err)
	assert.Equal("abcde", code.Code)
	assert.Equal("code", code.ResponseType)
	assert.Equal("http://john.doe.example.com", code.Me)
	assert.Equal("http://client.example.com", code.ClientID)
	assert.Equal("http://client.example.com/callback", code.RedirectURI)
	assert.WithinDuration(code.CreatedAt, now, 10*time.Millisecond)
	assert.False(code.Expired())
}

func TestCodeWithExpiry(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient)
	defer db.Close()

	now := time.Now().Add(codeExpiry)

	err := db.CreateSession(Session{
		ResponseType: "code",
		Me:           "http://john.doe.example.com",
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://client.example.com/callback",
		CreatedAt:    now,
	})
	assert.Nil(err)

	err = db.CreateCode("http://john.doe.example.com", "abcde", now)
	assert.Nil(err)

	code, err := db.Code("abcde")
	assert.Nil(err)
	assert.Equal("abcde", code.Code)
	assert.Equal("code", code.ResponseType)
	assert.Equal("http://john.doe.example.com", code.Me)
	assert.Equal("http://client.example.com", code.ClientID)
	assert.Equal("http://client.example.com/callback", code.RedirectURI)
	assert.WithinDuration(code.CreatedAt, now, 10*time.Millisecond)
	assert.True(code.Expired())
}

func TestCodeReadTwice(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient)
	defer db.Close()

	now := time.Now()

	err := db.CreateSession(Session{
		ResponseType: "code",
		Me:           "http://john.doe.example.com",
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://client.example.com/callback",
		CreatedAt:    now,
	})
	assert.Nil(err)

	err = db.CreateCode("http://john.doe.example.com", "abcde", now)
	assert.Nil(err)

	code, err := db.Code("abcde")
	assert.Nil(err)
	assert.Equal("abcde", code.Code)

	_, err = db.Code("abcde")
	assert.Equal(sql.ErrNoRows, err)
}
