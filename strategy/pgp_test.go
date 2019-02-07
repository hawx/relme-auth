package strategy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
	"hawx.me/code/assert"
)

func TestPGPMatch(t *testing.T) {
	pgp := PGP(new(fakeStore), "", id, http.DefaultClient)

	parsed, err := url.Parse("pgp")
	assert.Nil(t, err)
	assert.True(t, pgp.Match(parsed))
}

func TestPGPNotMatch(t *testing.T) {
	pgp := PGP(new(fakeStore), "", id, http.DefaultClient)

	testCases := []string{
		"what",
		"example.com",
		"https://example.com/somebody",
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			parsed, err := url.Parse(tc)
			assert.Nil(t, err)
			assert.False(t, pgp.Match(parsed))
		})
	}
}

func TestPGPAuthFlow(t *testing.T) {
	assert := assert.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	public, _ := os.Open("testdata/public.asc")
	defer public.Close()

	key := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(w, public)
	}))
	defer key.Close()

	state := "randomstatestring"
	expectedURL := key.URL

	store := &pgpStore{
		State: state,
	}

	pgp := &authPGP{
		AuthURL:    server.URL + "/oauth/authorize",
		ClientID:   id,
		Store:      store,
		httpClient: http.DefaultClient,
	}

	// 1. Redirect
	redirectURL, err := pgp.Redirect(expectedURL)
	assert.Nil(err)

	expectedRedirectURL := fmt.Sprintf("%s/oauth/authorize?challenge=%s&client_id=%s&state=%s", server.URL, store.Challenge, id, state)
	assert.Equal(expectedRedirectURL, redirectURL)

	// 2. Callback
	profileURL, err := pgp.Callback(url.Values{
		"state":  {state},
		"signed": {sign(store.Challenge, "testdata/private.asc")},
	})
	assert.Nil(err)
	assert.Equal(expectedURL, profileURL)
}

func TestPGPAuthFlowWithBadKey(t *testing.T) {
	assert := assert.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	public, _ := os.Open("testdata/public.asc")
	defer public.Close()

	key := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(w, public)
	}))
	defer key.Close()

	expectedURL := key.URL
	state := "randomstatestring"

	store := &pgpStore{
		State: state,
	}

	pgp := &authPGP{
		AuthURL:    server.URL + "/oauth/authorize",
		ClientID:   id,
		Store:      store,
		httpClient: http.DefaultClient,
	}

	// 1. Redirect
	redirectURL, err := pgp.Redirect(expectedURL)
	assert.Nil(err)

	expectedRedirectURL := fmt.Sprintf("%s/oauth/authorize?challenge=%s&client_id=%s&state=%s", server.URL, store.Challenge, id, state)
	assert.Equal(expectedRedirectURL, redirectURL)

	// 2. Callback
	profileURL, err := pgp.Callback(url.Values{
		"state":  {state},
		"signed": {sign("abcde", "testdata/other_private.asc")},
	})
	assert.Equal(ErrUnauthorized, err)
	assert.Equal("", profileURL)
}

type pgpStore struct {
	State     string
	Challenge string
	Link      string
}

func (s *pgpStore) Insert(link string) (state string, err error) {
	s.Link = link

	return s.State, nil
}

func (s *pgpStore) Set(key, value string) error {
	s.Link = key
	s.Challenge = value
	return nil
}

func (s *pgpStore) Claim(state string) (link string, ok bool) {
	if state == s.State {
		return s.Link, true
	}
	if state == s.Link {
		return s.Challenge, true
	}

	return "", false
}

func sign(challenge, key string) string {
	private, _ := os.Open(key)
	defer private.Close()
	keyRing, _ := openpgp.ReadArmoredKeyRing(private)

	signedMsg := bytes.NewBuffer(nil)
	dec, _ := clearsign.Encode(signedMsg, keyRing[0].PrivateKey, nil)
	dec.Write([]byte(challenge))
	dec.Close()

	return signedMsg.String()
}
