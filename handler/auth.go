package handler

import (
	"encoding/json"
	"net/http"

	"hawx.me/code/mux"
	"hawx.me/code/relme"
	"hawx.me/code/relme-auth/state"
	"hawx.me/code/relme-auth/strategy"
)

func Auth(authStore state.Store, strategies strategy.Strategies) http.Handler {
	return mux.Method{
		"GET":  authGet(authStore, strategies),
		"POST": authPost(authStore),
	}
}

func authGet(authStore state.Store, strategies strategy.Strategies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := &state.Session{
			Me:          r.FormValue("me"),
			ClientID:    r.FormValue("client_id"),
			RedirectURI: r.FormValue("redirect_uri"),
		}
		authStore.Save(session)

		verifiedLinks, _ := relme.FindVerified(session.Me)
		if chosenStrategy, _, ok := strategies.Find(verifiedLinks); ok {
			redirectURL, err := chosenStrategy.Redirect(session.Me)
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

type authPostResponse struct {
	Me string `json:"me"`
}

func authPost(authStore state.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")

		if code == "" {
			writeJsonError(w, "invalid_request", "Missing 'code' parameter", http.StatusBadRequest)
			return
		}

		session, ok := authStore.GetByCode(code)
		if !ok {
			writeJsonError(w, "invalid_request", "The code provided was not valid", http.StatusNotFound)
			return
		}

		if session.Expired() {
			writeJsonError(w, "invalid_request", "The auth code has expired (valid for 60 seconds)", http.StatusNotFound)
			return
		}

		if session.RedirectURI != r.FormValue("redirect_uri") {
			writeJsonError(w, "invalid_request", "The 'redirect_uri' parameter did not match", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(authPostResponse{
			Me: session.Me,
		})
	})
}

type jsonError struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func writeJsonError(w http.ResponseWriter, error string, description string, statusCode int) {
	json.NewEncoder(w).Encode(jsonError{
		Error:       error,
		Description: description,
	})
	w.WriteHeader(statusCode)
}
