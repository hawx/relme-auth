package sqlite

import (
	"testing"
	"time"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
)

func TestCode(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared")
	defer db.Close()

	now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	err := db.CreateCode(data.Code{
		Code:         "abcde",
		ResponseType: "code",
		Me:           "http://john.doe.example.com",
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://client.example.com/callback",
		CreatedAt:    now,
	})
	assert.Nil(err)

	code, err := db.Code("abcde")
	assert.Nil(err)
	assert.Equal("abcde", code.Code)
	assert.Equal("code", code.ResponseType)
	assert.Equal("http://john.doe.example.com", code.Me)
	assert.Equal("http://client.example.com", code.ClientID)
	assert.Equal("http://client.example.com/callback", code.RedirectURI)
	assert.Equal(code.CreatedAt, now)
}
