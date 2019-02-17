package handler

import (
	"log"
	"net/http"
	"net/url"
	"time"

	"hawx.me/code/relme-auth/data"
)

type continueStore interface {
	Login(*http.Request) (string, error)
	Session(string) (data.Session, error)
	CreateCode(me, code string, createdAt time.Time) error
}

// Continue handles a user choosing to authenticate using a previous session.
func Continue(store continueStore, generator func() (string, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userProfileURL, err := store.Login(r)
		if err != nil {
			log.Println(err)
			http.Error(w, "how did you get here?", http.StatusInternalServerError)
		}

		session, err := store.Session(userProfileURL)
		if err != nil {
			http.Error(w, "Who are you?", http.StatusInternalServerError)
			return
		}
		if session.Expired() {
			http.Error(w, "Auth session expired", http.StatusInternalServerError)
			return
		}

		code, err := generator()
		if err != nil {
			log.Println("handler/continue could not generate code:", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}
		if err = store.CreateCode(session.Me, code, time.Now()); err != nil {
			log.Println("handler/continue could not create code:", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		redirectURI, err := url.Parse(session.RedirectURI)
		if err != nil {
			log.Println("handler/continue could not parse redirect_uri:", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		query := redirectURI.Query()
		query.Set("code", code)
		query.Set("state", session.State)
		redirectURI.RawQuery = query.Encode()

		http.Redirect(w, r, redirectURI.String(), http.StatusFound)
	})
}
