package handler

import (
	"fmt"
	"net/http"
	"net/url"

	"hawx.me/code/mux"
	"hawx.me/code/relme"
	"hawx.me/code/relme-auth/store"
	"hawx.me/code/relme-auth/strategy"
)

func Choose(authStore store.SessionStore, strategies strategy.Strategies) http.Handler {
	return mux.Method{
		"GET": chooseProvider(authStore, strategies),
	}
}

func chooseProvider(authStore store.SessionStore, strategies strategy.Strategies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		me := r.FormValue("me")

		verifiedLinks, err := relme.FindVerified(me)
		if err != nil {
			http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
			return
		}

		found, ok := strategies.Allowed(verifiedLinks)
		if !ok {
			http.Error(w, "No rel=\"me\" links on your profile match a known provider", http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, `<!DOCTYPE html><html><body>

<p>Authenticate using one of the methods below to sign-in to %s as %s
<ul>`, r.FormValue("client_id"), me)

		for profileURL, strategy := range found {
			query := url.Values{
				"me":           {me},
				"provider":     {strategy.Name()},
				"profile":      {profileURL},
				"client_id":    {r.FormValue("client_id")},
				"redirect_uri": {r.FormValue("redirect_uri")},
			}

			fmt.Fprintf(w, `
<li>
  <a href="/auth/start?%s">%s - %s</a>
</li>
`, query.Encode(), strategy.Name(), profileURL)
		}

		fmt.Fprint(w, `</ul></body></html>`)
	})
}
