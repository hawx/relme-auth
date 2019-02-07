package sqlite

import (
	"testing"
	"time"

	"hawx.me/code/assert"
	"hawx.me/code/relme-auth/data"
)

func TestClient(t *testing.T) {
	assert := assert.New(t)

	db, _ := Open("file::memory:?mode=memory&cache=shared")
	defer db.Close()

	now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err := db.CacheClient(data.Client{
		ID:          "http://client.example.com",
		RedirectURI: "http://client.example.com/callback",
		Name:        "Example",
		UpdatedAt:   now,
	})
	assert.Nil(err)

	client, err := db.Client("http://client.example.com")
	assert.Nil(err)
	assert.Equal("http://client.example.com", client.ID)
	assert.Equal("http://client.example.com/callback", client.RedirectURI)
	assert.Equal("Example", client.Name)
	assert.Equal(now, client.UpdatedAt)
}
