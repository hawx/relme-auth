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
		var (
			grantType    = r.FormValue("grant_type")
			code         = r.FormValue("code")
			clientID     = r.FormValue("client_id")
			redirectURI  = r.FormValue("redirect_uri")
			codeVerifier = r.FormValue("code_verifier")
		)

		w.Header().Set("Content-Type", "application/json")

		if code == "" {
			writeJSONError(w, "invalid_request", "Missing 'code' parameter", http.StatusBadRequest)
			return
		}

		if grantType == "" {
			grantType = "authorization_code"
		}

		if grantType != "authorization_code" {
			writeJSONError(w, "invalid_request", "Only grant type of 'authorization_code' is supported", http.StatusBadRequest)
			return
		}

		session, err := store.Code(code)
		if err != nil || (session.ResponseType != "id" && session.ResponseType != "code") {
			writeJSONError(w, "invalid_request", "The code provided was not valid", http.StatusBadRequest)
			return
		}

		if session.Expired() {
			writeJSONError(w, "invalid_request", "The auth code has expired (valid for 60 seconds)", http.StatusBadRequest)
			return
		}

		if session.ClientID != data.ParseClientID(clientID) {
			writeJSONError(w, "invalid_request", "The 'client_id' parameter did not match", http.StatusBadRequest)
			return
		}
		if session.RedirectURI != redirectURI {
			writeJSONError(w, "invalid_request", "The 'redirect_uri' parameter did not match", http.StatusBadRequest)
			return
		}

		if session.ResponseType == "code" {
			ok, err := session.VerifyChallenge(codeVerifier)
			if err != nil {
				writeJSONError(w, "invalid_request", err.Error(), http.StatusBadRequest)
				return
			}
			if !ok {
				writeJSONError(w, "invalid_request", "Provided 'code_verifier' does not match initial challenge", http.StatusBadRequest)
				return
			}
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
