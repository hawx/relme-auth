package data

import (
	"database/sql"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestToken(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared")
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
	assert.Equal(token.CreatedAt, now)

	err = db.RevokeToken("abcde")
	assert.Nil(err)

	_, err = db.Token("abcde")
	assert.Equal(sql.ErrNoRows, err)

	err = db.RevokeToken("abcde")
	assert.Nil(err)
}
