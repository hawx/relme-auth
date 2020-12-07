package data

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestCode(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{Code: time.Hour})
	defer db.Close()

	now := time.Now()

	err := db.CreateSession(Session{
		ResponseType: "code",
		Me:           "http://john.doe.example.com",
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://client.example.com/callback",
		CreatedAt:    now,
	})
	assert(err).Nil()

	err = db.CreateCode("http://john.doe.example.com", "abcde", now)
	assert(err).Nil()

	code, err := db.Code("abcde")
	assert(err).Nil()
	assert(code.Code).Equal("abcde")
	assert(code.ResponseType).Equal("code")
	assert(code.Me).Equal("http://john.doe.example.com")
	assert(code.ClientID).Equal("http://client.example.com")
	assert(code.RedirectURI).Equal("http://client.example.com/callback")
	assert(code.CreatedAt).WithinDuration(now, 10*time.Millisecond)
	assert(code.Expired()).False()
}

func TestCodeWithExpiry(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{Code: time.Hour})
	defer db.Close()

	now := time.Now().Add(-time.Hour)

	err := db.CreateSession(Session{
		ResponseType: "code",
		Me:           "http://john.doe.example.com",
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://client.example.com/callback",
		CreatedAt:    now,
	})
	assert(err).Must.Nil()

	err = db.CreateCode("http://john.doe.example.com", "abcde", now)
	assert(err).Must.Nil()

	code, err := db.Code("abcde")
	assert(err).Must.Nil()
	assert(code.Code).Equal("abcde")
	assert(code.ResponseType).Equal("code")
	assert(code.Me).Equal("http://john.doe.example.com")
	assert(code.ClientID).Equal("http://client.example.com")
	assert(code.RedirectURI).Equal("http://client.example.com/callback")
	assert(code.CreatedAt).WithinDuration(now, 10*time.Millisecond)
	assert(code.Expired()).True()
}

func TestCodeReadTwice(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{})
	defer db.Close()

	now := time.Now()

	err := db.CreateSession(Session{
		ResponseType: "code",
		Me:           "http://john.doe.example.com",
		ClientID:     "http://client.example.com",
		RedirectURI:  "http://client.example.com/callback",
		CreatedAt:    now,
	})
	assert(err).Must.Nil()

	err = db.CreateCode("http://john.doe.example.com", "abcde", now)
	assert(err).Must.Nil()

	code, err := db.Code("abcde")
	assert(err).Must.Nil()
	assert(code.Code).Equal("abcde")

	_, err = db.Code("abcde")
	assert(err).Equal(sql.ErrNoRows)
}

func TestCodeVerifyChallenge(t *testing.T) {
	testCases := []struct {
		name      string
		method    string
		challenge string
		verifier  string
		ok        bool
		err       error
	}{
		{
			name:   "unknown method",
			method: "what",
			ok:     false,
			err:    ErrUnknownCodeChallengeMethod,
		},
		{
			name:      "plain matching",
			method:    "plain",
			challenge: "test",
			verifier:  "test",
			ok:        true,
		},
		{
			name:      "plain non-matching",
			method:    "plain",
			challenge: "test",
			verifier:  "nope",
			ok:        false,
		},
		{
			name:      "S256 matching",
			method:    "S256",
			challenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			verifier:  "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			ok:        true,
		},
		{
			name:      "S256 non-matching",
			method:    "S256",
			challenge: "test",
			verifier:  "nope",
			ok:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.Wrap(t)

			code := Code{
				CodeChallenge:       tc.challenge,
				CodeChallengeMethod: tc.method,
			}

			ok, err := code.VerifyChallenge(tc.verifier)
			assert(ok).Equal(tc.ok)
			assert(err).Equal(tc.err)
		})
	}
}
