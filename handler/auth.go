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
		"GET":  redirectToProvider(authStore, strategies),
		"POST": verifyCode(authStore),
	}
}

func redirectToProvider(authStore state.Store, strategies strategy.Strategies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		me := r.FormValue("me")

		verifiedLinks, err := relme.FindVerified(me)
		if err != nil {
			http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
			return
		}

		chosenStrategy, profileURI, ok := strategies.Find(verifiedLinks)
		if !ok {
			http.Error(w, "No rel=\"me\" links on your profile match a known provider", http.StatusBadRequest)
			return
		}

		redirectURL, err := chosenStrategy.Redirect(me)
		if err != nil {
			http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
			return
		}

		authStore.Save(&state.Session{
			Me:          me,
			ClientID:    r.FormValue("client_id"),
			RedirectURI: r.FormValue("redirect_uri"),
			Provider:    chosenStrategy.Name(),
			ProfileURI:  profileURI,
		})
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})
}

type verifyCodeResponse struct {
	Me string `json:"me"`
}

func verifyCode(authStore state.Store) http.Handler {
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

		json.NewEncoder(w).Encode(verifyCodeResponse{
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
