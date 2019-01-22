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
			writeJSONError(w, "invalid_request", "Missing 'code' parameter", http.StatusBadRequest)
			return
		}

		session, ok := authStore.GetByCode(code)
		if !ok {
			writeJSONError(w, "invalid_request", "The code provided was not valid", http.StatusBadRequest)
			return
		}

		if session.Expired() {
			writeJSONError(w, "invalid_request", "The auth code has expired (valid for 60 seconds)", http.StatusBadRequest)
			return
		}

		if session.ClientID != r.FormValue("client_id") {
			writeJSONError(w, "invalid_request", "The 'client_id' parameter did not match", http.StatusBadRequest)
			return
		}
		if session.RedirectURI != r.FormValue("redirect_uri") {
			writeJSONError(w, "invalid_request", "The 'redirect_uri' parameter did not match", http.StatusBadRequest)
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

func writeJSONError(w http.ResponseWriter, error string, description string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(jsonError{
		Error:       error,
		Description: description,
	})
}
