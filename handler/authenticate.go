package handler

import (
	"fmt"
	"net/http"
	"net/url"

	"hawx.me/code/relme"
	"hawx.me/code/relme-auth/state"
	"hawx.me/code/relme-auth/strategy"
)

func Authenticate(authStore state.Store, strategies []strategy.Strategy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(415)
			return
		}

		me := r.FormValue("me")

		verifiedLinks, _ := relme.FindVerified(me)
		if chosenStrategy, expectedLink, ok := findStrategy(verifiedLinks, strategies); ok {
			state, err := authStore.Insert(expectedLink)
			if err != nil {
				http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, chosenStrategy.Redirect(state), http.StatusFound)
			return
		}

		http.Redirect(w, r, "/no-strategies", http.StatusFound)
	})
}

func findStrategy(verifiedLinks []string, strategies []strategy.Strategy) (s strategy.Strategy, expectedLink string, ok bool) {
	for _, link := range verifiedLinks {
		fmt.Printf("me=%s\n", link)
		linkURL, _ := url.Parse(link)

		for _, strategy := range strategies {
			if strategy.Match(linkURL) {
				fmt.Printf("Can authenticate with %s\n", link)
				return strategy, link, true
			}
		}
	}

	return
}
