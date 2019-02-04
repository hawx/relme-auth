package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"hawx.me/code/mux"
	"hawx.me/code/relme-auth/data"
)

func Token(authStore data.SessionStore) http.Handler {
	return mux.Method{
		"POST": tokenEndpoint(authStore),
		"GET":  verifyTokenEndpoint(authStore),
	}
}

func tokenEndpoint(authStore data.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("action") == "revoke" {
			token := r.FormValue("token")
			if token != "" {
				authStore.RevokeByToken(token)
			}

			return
		}

		var (
			grantType   = r.FormValue("grant_type")
			code        = r.FormValue("code")
			clientID    = r.FormValue("client_id")
			redirectURI = r.FormValue("redirect_uri")
			me          = r.FormValue("me")
		)

		if grantType != "authorization_code" {
			writeJSONError(w, "invalid_request", "The grant_type is not understood", http.StatusBadRequest)
			return
		}

		session, ok := authStore.GetByCode(code)
		if !ok || session.ResponseType != "code" {
			writeJSONError(w, "invalid_request", "The code provided was not valid", http.StatusBadRequest)
			return
		}

		if session.Expired() {
			writeJSONError(w, "invalid_request", "The auth code has expired (valid for 60 seconds)", http.StatusBadRequest)
			return
		}

		if session.ClientID != clientID {
			writeJSONError(w, "invalid_request", "The 'client_id' parameter did not match", http.StatusBadRequest)
			return
		}
		if session.RedirectURI != redirectURI {
			writeJSONError(w, "invalid_request", "The 'redirect_uri' parameter did not match", http.StatusBadRequest)
			return
		}
		if session.Me != me {
			writeJSONError(w, "invalid_request", "The 'me' parameter did not match", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: session.Token,
			TokenType:   "Bearer",
			Scope:       strings.Join(session.Scopes, " "),
			Me:          session.Me,
		})
	}
}

func verifyTokenEndpoint(authStore data.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authParts := strings.Fields(r.Header.Get("Authorization"))

		if len(authParts) != 2 || authParts[0] != "Bearer" {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		session, ok := authStore.GetByToken(authParts[1])
		if !ok {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		json.NewEncoder(w).Encode(tokenVerificationResponse{
			Me:       session.Me,
			ClientID: session.ClientID,
			Scope:    strings.Join(session.Scopes, " "),
		})
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Me          string `json:"me"`
}

type tokenVerificationResponse struct {
	Me       string `json:"me"`
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
}
