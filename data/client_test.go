package data

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestClient(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient)
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
	assert.Nil(err)
	assert.Equal(s.URL, client.ID)
	assert.Equal(s.URL+"/callback", client.RedirectURI)
	assert.Equal("My App", client.Name)
	assert.WithinDuration(time.Now(), time.Now(), time.Second)
	assert.Equal(1, callCount)

	client, err = db.Client(s.URL, s.URL+"/callback")
	assert.Nil(err)
	assert.Equal(s.URL, client.ID)
	assert.Equal(s.URL+"/callback", client.RedirectURI)
	assert.Equal("My App", client.Name)
	assert.WithinDuration(time.Now(), time.Now(), time.Second)
	assert.Equal(1, callCount)

	// new callback means new call to clientID
	client, err = db.Client(s.URL, s.URL+"/not-callback")
	assert.Nil(err)
	assert.Equal(s.URL, client.ID)
	assert.Equal(s.URL+"/not-callback", client.RedirectURI)
	assert.Equal("My App", client.Name)
	assert.WithinDuration(time.Now(), time.Now(), time.Second)
	assert.Equal(2, callCount)

	// old callback was forgotten, so this calls clientID
	client, err = db.Client(s.URL, s.URL+"/callback")
	assert.Nil(err)
	assert.Equal(s.URL, client.ID)
	assert.Equal(s.URL+"/callback", client.RedirectURI)
	assert.Equal("My App", client.Name)
	assert.WithinDuration(time.Now(), time.Now(), time.Second)
	assert.Equal(3, callCount)
}
