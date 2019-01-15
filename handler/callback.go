package handler

import (
	"net/http"

	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/strategy"
)

// Callback handles the return from the authentication provider by delegating to
// the relevant strategy. If authentication was successful, and for the correct
// user, then it will redirect to the "redirect_uri" that the authentication
// flow was originally started with. A "code" parameter is returned which can be
// verified as belonging to the authenticated user for a short period of time.
func Callback(authStore data.SessionStore, strat strategy.Strategy) http.Handler {
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
