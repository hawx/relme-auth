package data

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/sessions"

	"hawx.me/code/assert"
)

type fakeCookieStore struct{}

func (f *fakeCookieStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return nil, nil
}
func (f *fakeCookieStore) New(r *http.Request, name string) (*sessions.Session, error) {
	return nil, nil
}
func (f *fakeCookieStore) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	return nil
}

func TestForget(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{})
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

	err = db.CacheProfile(Profile{
		Me:        "http://john.doe.example.com",
		UpdatedAt: now,
		Methods: []Method{
			{Provider: "someone", Profile: "http://someone.example.com/john.doe"},
			{Provider: "else", Profile: "http://else.example.com/john.doe"},
			{Provider: "other", Profile: "http://other.example.com/john.doe"},
		},
	})
	assert.Nil(err)

	err = db.CreateToken(Token{
		Token:     "abcde",
		Me:        "http://john.doe.example.com",
		ClientID:  "http://client.example.com",
		Scope:     "create media",
		CreatedAt: now,
	})
	assert.Nil(err)

	err = db.Forget("http://john.doe.example.com")
	assert.Nil(err)

	_, err = db.Session("http://john.doe.example.com")
	assert.Equal(sql.ErrNoRows, err)

	_, err = db.Profile("http://john.doe.example.com")
	assert.Equal(sql.ErrNoRows, err)

	tokens, _ := db.Tokens("https://john.doe.example.com")
	assert.Len(tokens, 0)
}
