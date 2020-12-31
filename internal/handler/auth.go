package handler

import (
	"net/http"

	"hawx.me/code/relme-auth/internal/data"
	"hawx.me/code/relme-auth/internal/strategy"
)

type authStore interface {
	Session(string) (data.Session, error)
	SetProvider(me, provider, profileURI string) error
}

// Auth takes the chosen provider and initiates authentication by redirecting
// the user to the 3rd party. It takes a number of parameters:
//
//   - me: URL originally entered of who we are trying to authenticate
//   - provider: 3rd party authentication provider that was chosen
//   - profile: URL expected to be matched by the provider
//   - redirect_uri: final URI to redirect to when auth is finished
func Auth(store authStore, strategies strategy.Strategies, httpClient *http.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			me          = r.FormValue("me")
			provider    = r.FormValue("provider")
			profile     = r.FormValue("profile")
			redirectURI = r.FormValue("redirect_uri")

			chosenStrategy strategy.Strategy
			ok             bool
		)

		session, err := store.Session(me)
		if err != nil {
			http.Error(w, "you need to start at the start", http.StatusBadRequest)
			return
		}
		if session.Expired() {
			http.Error(w, "auth session expired", http.StatusBadRequest)
			return
		}
		if session.RedirectURI != redirectURI {
			http.Error(w, "redirect_uri is untrustworthy", http.StatusBadRequest)
			return
		}

		for _, s := range strategies {
			if s.Name() == provider {
				chosenStrategy = s
				ok = true
				break
			}
		}

		if !ok {
			http.Error(w, "No rel=\"me\" links on your profile match a known provider", http.StatusBadRequest)
			return
		}

		redirectURL, err := chosenStrategy.Redirect(me, profile)
		if err != nil {
			http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
			return
		}

		session.Provider = provider
		session.ProfileURI = profile

		store.SetProvider(session.Me, provider, profile)

		http.Redirect(w, r, redirectURL, http.StatusFound)
	})
}
