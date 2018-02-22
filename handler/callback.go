package handler

import (
	"crypto/rsa"
	"fmt"
	"net/http"

	"hawx.me/code/relme-auth/state"
	"hawx.me/code/relme-auth/strategy"
	"hawx.me/code/relme-auth/token"
)

func Callback(privateKey *rsa.PrivateKey, authStore state.Store, strat strategy.Strategy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		state := r.FormValue("state")

		expectedURL, ok := authStore.Claim(state)
		if !ok {
			http.Error(w, "How did you get here?", http.StatusInternalServerError)
			return
		}

		userProfileURL, err := strat.Callback(code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if userProfileURL != expectedURL {
			http.Error(w, "You are not the user I was expecting", http.StatusUnauthorized)
			return
		}

		jwt, _ := token.NewJWT(expectedURL).Encode(privateKey)
		fmt.Fprint(w, jwt)
	})
}
