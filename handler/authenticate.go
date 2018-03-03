package handler

import (
	"net/http"

	"hawx.me/code/relme"
	"hawx.me/code/relme-auth/state"
	"hawx.me/code/relme-auth/strategy"
)

func Authenticate(authStore state.Store, strategies strategy.Strategies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(415)
			return
		}

		me := r.FormValue("me")

		verifiedLinks, _ := relme.FindVerified(me)
		if chosenStrategy, _, ok := strategies.Find(verifiedLinks); ok {
			redirectURL, err := chosenStrategy.Redirect(me)
			if err != nil {
				http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, redirectURL, http.StatusFound)
			return
		}

		http.Redirect(w, r, "/no-strategies", http.StatusFound)
	})
}
