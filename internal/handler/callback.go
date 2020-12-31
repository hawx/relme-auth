package handler

import (
	"log"
	"net/http"
	"net/url"
	"time"

	"hawx.me/code/relme-auth/internal/data"
	"hawx.me/code/relme-auth/internal/strategy"
)

type callbackStore interface {
	SaveLogin(http.ResponseWriter, *http.Request, string) error
	Session(string) (data.Session, error)
	CreateCode(me, code string, createdAt time.Time) error
}

// Callback handles the return from the authentication provider by delegating to
// the relevant strategy. If authentication was successful, and for the correct
// user, then it will redirect to the "redirect_uri" that the authentication
// flow was originally started with. A "code" parameter is returned which can be
// verified as belonging to the authenticated user for a short period of time.
func Callback(store callbackStore, strat strategy.Strategy, generator func() (string, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Println("handler/callback failed to parse form: ", err)
			http.Error(w, "the request was bad", http.StatusBadRequest)
			return
		}

		userProfileURL, err := strat.Callback(r.Form)
		if err != nil {
			if err == strategy.ErrUnauthorized {
				http.Error(w, "the chosen provider says you are unauthorized", http.StatusUnauthorized)
			} else {
				log.Println("handler/callback unknown error: ", err)
				http.Error(w, "something went wrong with the chosen provider, maybe try again with a different choice?", http.StatusInternalServerError)
			}
			return
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
			log.Println("handler/callback could not generate code:", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}
		if err = store.CreateCode(session.Me, code, time.Now()); err != nil {
			log.Println("handler/callback could not create code:", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		redirectURI, err := url.Parse(session.RedirectURI)
		if err != nil {
			log.Println("handler/callback could not parse redirect_uri:", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		query := redirectURI.Query()
		query.Set("code", code)
		query.Set("state", session.State)
		redirectURI.RawQuery = query.Encode()

		store.SaveLogin(w, r, session.Me)

		http.Redirect(w, r, redirectURI.String(), http.StatusFound)
	})
}
