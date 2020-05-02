package data

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/sessions"

	"hawx.me/code/assert"
)

func TestLogin(t *testing.T) {
	assert := assert.Wrap(t)

	cookies := sessions.NewCookieStore([]byte("hey"))

	db, _ := Open("file::memory:?mode=memory&cache=shared", http.DefaultClient, cookies, Expiry{Login: time.Hour})
	defer db.Close()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			err := db.SaveLogin(w, r, "http://its.me/")
			assert(err).Must.Nil()
		} else {
			me, err := db.Login(r)
			assert(err).Must.Nil()
			assert(me).Must.Equal("http://its.me/")
		}
	}))
	defer s.Close()

	resp, err := http.Post(s.URL, "", nil)
	assert(err).Must.Nil()

	req, _ := http.NewRequest("GET", s.URL, nil)
	req.AddCookie(resp.Cookies()[0])

	_, err = http.DefaultClient.Do(req)
	assert(err).Must.Nil()
}
