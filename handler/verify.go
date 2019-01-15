package handler

import (
	"encoding/json"
	"net/http"

	"hawx.me/code/relme-auth/data"
)

// Verify allows clients to check who a particular "code" belongs to, or whether
// it is invalid.
func Verify(authStore data.SessionStore) http.Handler {
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

type verifyCodeResponse struct {
	Me string `json:"me"`
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
