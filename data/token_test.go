package data

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestToken(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{})
	defer db.Close()

	now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	err := db.CreateToken(Token{
		Token:     "abcde",
		Me:        "http://john.doe.example.com",
		ClientID:  "http://client.example.com",
		Scope:     "create media",
		CreatedAt: now,
	})
	assert.Nil(err)

	token, err := db.Token("abcde")
	assert.Nil(err)
	assert.Equal("abcde", token.Token)
	assert.Equal("http://john.doe.example.com", token.Me)
	assert.Equal("http://client.example.com", token.ClientID)
	assert.Equal("create media", token.Scope)
	assert.Equal(now, token.CreatedAt)

	tokens, err := db.Tokens("http://john.doe.example.com")
	assert.Nil(err)
	if assert.Len(tokens, 1) {
		assert.Equal("abcde", tokens[0].Token)
		assert.Equal("http://john.doe.example.com", tokens[0].Me)
		assert.Equal("http://client.example.com", tokens[0].ClientID)
		assert.Equal("create media", tokens[0].Scope)
		assert.Equal(now, tokens[0].CreatedAt)
	}

	err = db.RevokeToken("abcde")
	assert.Nil(err)

	_, err = db.Token("abcde")
	assert.Equal(sql.ErrNoRows, err)

	tokens, err = db.Tokens("http://john.doe.example.com")
	assert.Nil(err)
	assert.Len(tokens, 0)

	err = db.RevokeToken("abcde")
	assert.Nil(err)
}

func TestTokenRevokeByClient(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{})
	defer db.Close()

	now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	err := db.CreateToken(Token{
		Token:     "abcde",
		Me:        "http://john.doe.example.com",
		ClientID:  "http://client.example.com",
		Scope:     "create media",
		CreatedAt: now,
	})
	assert.Nil(err)

	token, err := db.Token("abcde")
	assert.Nil(err)
	assert.Equal("abcde", token.Token)
	assert.Equal("http://john.doe.example.com", token.Me)
	assert.Equal("http://client.example.com", token.ClientID)
	assert.Equal("create media", token.Scope)
	assert.Equal(now, token.CreatedAt)

	tokens, err := db.Tokens("http://john.doe.example.com")
	assert.Nil(err)
	if assert.Len(tokens, 1) {
		assert.Equal("abcde", tokens[0].Token)
		assert.Equal("http://john.doe.example.com", tokens[0].Me)
		assert.Equal("http://client.example.com", tokens[0].ClientID)
		assert.Equal("create media", tokens[0].Scope)
		assert.Equal(now, tokens[0].CreatedAt)
	}

	err = db.RevokeClient("http://john.doe.example.com", "http://client.example.com")
	assert.Nil(err)

	_, err = db.Token("abcde")
	assert.Equal(sql.ErrNoRows, err)

	err = db.RevokeClient("http://john.doe.example.com", "http://client.example.com")
	assert.Nil(err)
}
