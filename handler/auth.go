package handler

import (
	"log"
	"net/http"

	"hawx.me/code/relme-auth/store"
	"hawx.me/code/relme-auth/strategy"
)

func Auth(authStore store.SessionStore, strategies strategy.Strategies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			me             = r.FormValue("me")
			provider       = r.FormValue("provider")
			profile        = r.FormValue("profile")
			clientID       = r.FormValue("client_id")
			redirectURI    = r.FormValue("redirect_uri")
			chosenStrategy strategy.Strategy
			ok             bool
		)

		for _, s := range strategies {
			if s.Name() == provider {
				chosenStrategy = s
				ok = true
			}
		}

		if !ok {
			http.Error(w, "No rel=\"me\" links on your profile match a known provider", http.StatusBadRequest)
			return
		}
		log.Println("Authenticating", me, "using", provider)

		redirectURL, err := chosenStrategy.Redirect(me)
		if err != nil {
			http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
			return
		}

		authStore.Save(&store.Session{
			Me:          me,
			ClientID:    clientID,
			RedirectURI: redirectURI,
			Provider:    provider,
			ProfileURI:  profile,
		})
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})
}
