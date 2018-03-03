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
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		userProfileURL, err := strat.Callback(r.Form)
		if err != nil {
			if err == strategy.ErrUnauthorized {
				http.Error(w, err.Error(), http.StatusUnauthorized)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		jwt, _ := token.NewJWT(userProfileURL).Encode(privateKey)
		fmt.Fprint(w, jwt)
	})
}
