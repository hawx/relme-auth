package handler

import (
	"crypto/rsa"
	"net/http"

	"hawx.me/code/relme-auth/store"
	"hawx.me/code/relme-auth/strategy"
)

func Callback(privateKey *rsa.PrivateKey, authStore store.SessionStore, strat strategy.Strategy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "form: "+err.Error(), http.StatusInternalServerError)
			return
		}

		userProfileURL, err := strat.Callback(r.Form)
		if err != nil {
			if err == strategy.ErrUnauthorized {
				http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
			} else {
				http.Error(w, "something: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		session, ok := authStore.Get(userProfileURL)
		if !ok {
			http.Error(w, "Who are you?", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, session.RedirectURI+"?code="+session.Code, http.StatusFound)
	})
}
