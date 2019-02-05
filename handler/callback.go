package handler

import (
	"log"
	"net/http"
	"net/url"

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

		session, ok := authStore.Get(userProfileURL)
		if !ok {
			http.Error(w, "Who are you?", http.StatusInternalServerError)
			return
		}

		query := url.Values{
			"code":  {session.Code},
			"state": {session.State},
		}

		http.Redirect(w, r, session.RedirectURI+"?"+query.Encode(), http.StatusFound)
	})
}
