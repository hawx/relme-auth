package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"hawx.me/code/relme-auth/data"
)

type verifyStore interface {
	Code(string) (data.Code, error)
}

// Verify allows clients to check who a particular "code" belongs to, or whether
// it is invalid.
func Verify(store verifyStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")

		if code == "" {
			writeJSONError(w, "invalid_request", "Missing 'code' parameter", http.StatusBadRequest)
			return
		}

		session, err := store.Code(code)
		if err != nil || session.ResponseType != "id" {
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

		if err := json.NewEncoder(w).Encode(verifyCodeResponse{
			Me: session.Me,
		}); err != nil {
			log.Println("handler/verify failed to write response:", err)
		}
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
	if err := json.NewEncoder(w).Encode(jsonError{
		Error:       error,
		Description: description,
	}); err != nil {
		log.Println("handler/verify failed to write response:", err)
	}
}
