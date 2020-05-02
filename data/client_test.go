package data

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestClient(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{Client: time.Hour})
	defer db.Close()

	callCount := 0

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte(`<div class="h-x-app">
  <h1 class="p-name">My App</h1>
</div>`))
	}))
	defer s.Close()

	client, err := db.Client(s.URL, s.URL+"/callback")
	assert(err).Must.Nil()
	assert(client.ID).Equal(s.URL)
	assert(client.RedirectURI).Equal(s.URL + "/callback")
	assert(client.Name).Equal("My App")
	assert(client.UpdatedAt).WithinDuration(time.Now(), time.Second)
	assert(client.expiresAt).WithinDuration(time.Now().Add(time.Hour), time.Second)
	assert(callCount).Equal(1)

	client, err = db.Client(s.URL, s.URL+"/callback")
	assert(err).Nil()
	assert(client.ID).Equal(s.URL)
	assert(client.RedirectURI).Equal(s.URL + "/callback")
	assert(client.Name).Equal("My App")
	assert(client.UpdatedAt).WithinDuration(time.Now(), time.Second)
	assert(client.expiresAt).WithinDuration(time.Now().Add(time.Hour), time.Second)
	assert(callCount).Equal(1)

	// new callback means new call to clientID
	client, err = db.Client(s.URL, s.URL+"/not-callback")
	assert(err).Nil()
	assert(client.ID).Equal(s.URL)
	assert(client.RedirectURI).Equal(s.URL + "/not-callback")
	assert(client.Name).Equal("My App")
	assert(client.UpdatedAt).WithinDuration(time.Now(), time.Second)
	assert(client.expiresAt).WithinDuration(time.Now().Add(time.Hour), time.Second)
	assert(callCount).Equal(2)

	// old callback was forgotten, so this calls clientID
	client, err = db.Client(s.URL, s.URL+"/callback")
	assert(err).Nil()
	assert(client.ID).Equal(s.URL)
	assert(client.RedirectURI).Equal(s.URL + "/callback")
	assert(client.Name).Equal("My App")
	assert(client.UpdatedAt).WithinDuration(time.Now(), time.Second)
	assert(client.expiresAt).WithinDuration(time.Now().Add(time.Hour), time.Second)
	assert(callCount).Equal(3)
}

func TestClientWhenLocalhost(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{Client: time.Hour})
	defer db.Close()

	client, err := db.Client("http://localhost:8080/", "http://localhost:8080/callback")
	assert(err).Nil()
	assert(client.ID).Equal("http://localhost:8080/")
	assert(client.RedirectURI).Equal("http://localhost:8080/callback")
	assert(client.Name).Equal("Local App")
	assert(client.UpdatedAt).WithinDuration(time.Now(), time.Second)
	assert(client.expiresAt).WithinDuration(time.Now().Add(time.Hour), time.Second)
}

func TestClientWithMismatchedRedirectURI(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{Client: time.Hour})
	defer db.Close()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<div class="h-x-app">
  <h1 class="p-name">My App</h1>
</div>`))
	}))
	defer s.Close()

	_, err := db.Client(s.URL, "http://example.com/callback")
	assert(err).NotNil()
}

func TestClientWithWhitelistedMismatchedRedirectURI(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{Client: time.Hour})
	defer db.Close()

	callCount := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte(`<link rel="redirect_uri" href="http://example.com/callback" />
<div class="h-x-app">
  <h1 class="p-name">My App</h1>
</div>`))
	}))
	defer s.Close()

	client, err := db.Client(s.URL, "http://example.com/callback")
	assert(err).Nil()
	assert(client.ID).Equal(s.URL)
	assert(client.RedirectURI).Equal("http://example.com/callback")
	assert(client.Name).Equal("My App")
	assert(client.UpdatedAt).WithinDuration(time.Now(), time.Second)
	assert(client.expiresAt).WithinDuration(time.Now().Add(time.Hour), time.Second)
	assert(callCount).Equal(1)
}

func TestClientWithWhitelistedMismatchedRedirectURIInHeader(t *testing.T) {
	assert := assert.Wrap(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, &fakeCookieStore{}, Expiry{Client: time.Hour})
	defer db.Close()

	callCount := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Link", `<http://example.com/callback>; rel="redirect_uri"`)
		w.Write([]byte(`<div class="h-x-app">
  <h1 class="p-name">My App</h1>
</div>`))
	}))
	defer s.Close()

	client, err := db.Client(s.URL, "http://example.com/callback")
	assert(err).Nil()
	assert(client.ID).Equal(s.URL)
	assert(client.RedirectURI).Equal("http://example.com/callback")
	assert(client.Name).Equal("My App")
	assert(client.UpdatedAt).WithinDuration(time.Now(), time.Second)
	assert(client.expiresAt).WithinDuration(time.Now().Add(time.Hour), time.Second)
	assert(callCount).Equal(1)
}
