package data

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestToken(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{})
	defer db.Close()

	now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	err := db.CreateToken(Token{
		ShortToken:    "abcde",
		LongTokenHash: "Ngi8oeROpsTSaOttsCJgJpiSwLQrhrvx53pvoWw8koI",
		Me:            "http://john.doe.example.com",
		ClientID:      "http://client.example.com",
		Scope:         "create media",
		CreatedAt:     now,
	})
	assert(err).Nil()

	token, err := db.Token("relmeauth_abcde_xyz")
	assert(err).Nil()
	assert(token.ShortToken).Equal("abcde")
	assert(token.Me).Equal("http://john.doe.example.com")
	assert(token.ClientID).Equal("http://client.example.com")
	assert(token.Scope).Equal("create media")
	assert(token.CreatedAt).Equal(now)

	tokens, err := db.Tokens("http://john.doe.example.com")
	assert(err).Nil()
	if assert(tokens).Len(1) {
		assert(tokens[0].ShortToken).Equal("abcde")
		assert(tokens[0].Me).Equal("http://john.doe.example.com")
		assert(tokens[0].ClientID).Equal("http://client.example.com")
		assert(tokens[0].Scope).Equal("create media")
		assert(tokens[0].CreatedAt).Equal(now)
	}

	err = db.RevokeToken("abcde")
	assert(err).Nil()

	_, err = db.Token("relmeauth_abcde_xyz")
	assert(err).Equal(sql.ErrNoRows)

	tokens, err = db.Tokens("http://john.doe.example.com")
	assert(err).Nil()
	assert(tokens).Len(0)

	err = db.RevokeToken("abcde")
	assert(err).Nil()
}

func TestTokenRevokeByClient(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{})
	defer db.Close()

	now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	err := db.CreateToken(Token{
		ShortToken:    "abcde",
		LongTokenHash: "Ngi8oeROpsTSaOttsCJgJpiSwLQrhrvx53pvoWw8koI",
		Me:            "http://john.doe.example.com",
		ClientID:      "http://client.example.com",
		Scope:         "create media",
		CreatedAt:     now,
	})
	assert(err).Nil()

	token, err := db.Token("relmeauth_abcde_xyz")
	assert(err).Nil()
	assert(token.ShortToken).Equal("abcde")
	assert(token.Me).Equal("http://john.doe.example.com")
	assert(token.ClientID).Equal("http://client.example.com")
	assert(token.Scope).Equal("create media")
	assert(token.CreatedAt).Equal(now)

	tokens, err := db.Tokens("http://john.doe.example.com")
	assert(err).Nil()
	if assert(tokens).Len(1) {
		assert(tokens[0].ShortToken).Equal("abcde")
		assert(tokens[0].Me).Equal("http://john.doe.example.com")
		assert(tokens[0].ClientID).Equal("http://client.example.com")
		assert(tokens[0].Scope).Equal("create media")
		assert(tokens[0].CreatedAt).Equal(now)
	}

	err = db.RevokeClient("http://john.doe.example.com", "http://client.example.com")
	assert(err).Nil()

	_, err = db.Token("relmeauth_abcde_xyz")
	assert(err).Equal(sql.ErrNoRows)

	err = db.RevokeClient("http://john.doe.example.com", "http://client.example.com")
	assert(err).Nil()
}
