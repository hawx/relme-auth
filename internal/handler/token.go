package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"hawx.me/code/mux"
	"hawx.me/code/relme-auth/internal/data"
)

type TokenDB interface {
	Code(string) (data.Code, error)
	Token(string) (data.Token, error)
	CreateToken(data.Token) error
	RevokeToken(string) error
}

func Token(store TokenDB, generator func(int) (string, error)) http.Handler {
	return mux.Method{
		"POST": tokenEndpoint(store, generator),
		"GET":  verifyTokenEndpoint(store),
	}
}

func tokenEndpoint(store TokenDB, generator func(int) (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("action") == "revoke" {
			token := r.FormValue("token")
			if token != "" {
				store.RevokeToken(token)
			}

			return
		}

		var (
			grantType    = r.FormValue("grant_type")
			code         = r.FormValue("code")
			clientID     = r.FormValue("client_id")
			redirectURI  = r.FormValue("redirect_uri")
			codeVerifier = r.FormValue("code_verifier")
		)

		if grantType != "authorization_code" {
			writeJSONError(w, "invalid_request", "The grant_type is not understood", http.StatusBadRequest)
			return
		}

		theCode, err := store.Code(code)
		if err != nil || theCode.ResponseType != "code" {
			writeJSONError(w, "invalid_request", "The code provided was not valid", http.StatusBadRequest)
			return
		}

		if theCode.Expired() {
			writeJSONError(w, "invalid_request", "The auth code has expired (valid for 60 seconds)", http.StatusBadRequest)
			return
		}

		if theCode.ClientID != data.ParseClientID(clientID) {
			writeJSONError(w, "invalid_request", "The 'client_id' parameter did not match", http.StatusBadRequest)
			return
		}
		if theCode.RedirectURI != redirectURI {
			writeJSONError(w, "invalid_request", "The 'redirect_uri' parameter did not match", http.StatusBadRequest)
			return
		}

		if theCode.CodeChallenge != "" {
			ok, err := theCode.VerifyChallenge(codeVerifier)
			if err != nil {
				writeJSONError(w, "invalid_request", err.Error(), http.StatusBadRequest)
				return
			}
			if !ok {
				writeJSONError(w, "invalid_request", "Provided 'code_verifier' does not match initial challenge", http.StatusBadRequest)
				return
			}
		} else if codeVerifier != "" {
			writeJSONError(w, "invalid_request", "Provided 'code_verifier' but initial request did not contain a challenge", http.StatusBadRequest)
			return
		}

		if len(theCode.Scope) == 0 {
			writeJSONError(w, "invalid_request", "Scopeless code must be exchanged using authorization endpoint", http.StatusBadRequest)
			return
		}

		token, tokenString, err := data.NewToken(generator, theCode)
		if err != nil {
			log.Println("handler/token could not generate token:", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		if err := store.CreateToken(token); err != nil {
			log.Println("handler/token could not persist token:", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: tokenString,
			TokenType:   "Bearer",
			Scope:       token.Scope,
			Me:          token.Me,
		})
	}
}

func verifyTokenEndpoint(store TokenDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authParts := strings.Fields(r.Header.Get("Authorization"))

		if len(authParts) != 2 || authParts[0] != "Bearer" {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		token, err := store.Token(authParts[1])
		if err != nil {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenVerificationResponse{
			Me:       token.Me,
			ClientID: token.ClientID,
			Scope:    token.Scope,
		})
	}
}

type meResponse struct {
	Me string `json:"me"`
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
