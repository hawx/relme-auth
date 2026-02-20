package handler

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"hawx.me/code/relme-auth/internal/cbor"
	"hawx.me/code/relme-auth/internal/data"
)

type CredentialDB interface {
	CreateCredential(me, credentialID, publicKey string, createdAt time.Time) error
	Credentials(me string) ([]data.Credential, error)
	SaveLogin(http.ResponseWriter, *http.Request, string) error
	Session(string) (data.Session, error)
	CreateCode(me, code string, createdAt time.Time) error
}

type strategyStore interface {
	strategyClaimStore
	strategySetStore
}

// TODO: where to put this? This is kind of a strategy, but kind of not
func Passkey(store strategyStore, db CredentialDB, generator func() (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const userProfileURL = "https://hawx.me/"

		switch r.Method {
		case http.MethodGet:
			// credential, err := db.Credential(r.FormValue("me"))
			credentials, err := db.Credentials(userProfileURL)
			if err != nil {
				// TODO: don't obviously leave this like this
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// TODO: handle multiple
			credential := credentials[0]

			buf := make([]byte, 32)
			rand.Reader.Read(buf)

			json.NewEncoder(w).Encode(struct {
				CredentialID string `json:"credentialID"`
				Challenge    string `json:"challenge"`
			}{
				CredentialID: credential.CredentialID,
				Challenge:    base64.StdEncoding.EncodeToString(buf),
			})

			store.Set(base64.RawURLEncoding.EncodeToString(buf), userProfileURL) // TODO: get this value from the request

		case http.MethodPost:
			var v PublicKeyCredential
			if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
				http.Error(w, "problem understanding what to do", http.StatusBadRequest)
				log.Println(err)
				return
			}

			clientData := v.Response.ClientDataJSON.V

			if clientData.Type != "webauthn.get" || clientData.Origin != "http://localhost:8080" {
				http.Error(w, "how did you get here?", http.StatusBadRequest)
				return
			}

			found, ok := store.Claim(clientData.Challenge)
			if !ok {
				http.Error(w, "no challenge found", http.StatusBadRequest)
				return
			}
			me := found.(string)

			// could also check that v.ID == credential.CredentialID
			//
			// technically we should be querying the DB on that, as we normally
			// wouldn't know who the user is. But this is a weird use of webauthn...
			credentials, err := db.Credentials(me)
			if err != nil {
				http.Error(w, "could not find credential", http.StatusBadRequest)
				log.Println(err)
				return
			}

			var credential data.Credential
			for _, cred := range credentials {
				if cred.CredentialID == v.ID {
					credential = cred
					break
				}
			}

			if credential.CredentialID == "" {
				http.Error(w, "credential not found", http.StatusBadRequest)
				return
			}

			authenticatorData, _ := base64.RawURLEncoding.DecodeString(v.Response.AuthenticatorData)
			if authenticatorData[32]&1 != 1 {
				http.Error(w, "user did not approve", http.StatusBadRequest)
				return
			}

			if bytes.Equal(authenticatorData[:32], []byte("localhost")) {
				http.Error(w, "wrong rpID", http.StatusBadRequest)
				return
			}

			keyData, _ := base64.StdEncoding.DecodeString(credential.PublicKey)

			// for my current combination it seems we have a COSE key, so proceed on that assumption
			decodedKeyData, _ := cbor.DecodeBytes(keyData)
			keyMap := decodedKeyData.(map[any]any)

			if keyMap[1].(int) != 2 || // check key type is elliptic curve
				keyMap[3].(int) != -7 || // check alg is ECDSA P-256
				keyMap[-1].(int) != 1 || // check curve is P-256
				len(keyMap[-2].([]byte)) != 32 || // check x has 32 bytes
				len(keyMap[-3].([]byte)) != 32 { // check y has 32 bytes
				http.Error(w, "key is not what I want", http.StatusBadRequest)
				return
			}

			x := new(big.Int)
			x.SetBytes(keyMap[-2].([]byte))

			y := new(big.Int)
			y.SetBytes(keyMap[-3].([]byte))

			key := &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     x,
				Y:     y,
			}

			clientDataHash := sha256.New()
			clientDataHash.Write(v.Response.ClientDataJSON.Bytes())

			hashData := sha256.New()
			hashData.Write(append(authenticatorData, clientDataHash.Sum(nil)...))

			sigData, _ := base64.RawURLEncoding.DecodeString(v.Response.Signature)

			if !ecdsa.VerifyASN1(key, hashData.Sum(nil), sigData) {
				http.Error(w, "could not verify", http.StatusBadRequest)
				log.Println(keyMap, "could not verify")
				return
			}

			// now do all the session stuff

			session, err := db.Session(userProfileURL)
			if err != nil {
				http.Error(w, "Who are you?", http.StatusInternalServerError)
				return
			}
			if session.Expired() {
				http.Error(w, "Auth session expired", http.StatusInternalServerError)
				return
			}

			code, err := generator()
			if err != nil {
				log.Println("handler/callback could not generate code:", err)
				http.Error(w, "something went wrong", http.StatusInternalServerError)
				return
			}
			if err = db.CreateCode(session.Me, code, time.Now()); err != nil {
				log.Println("handler/callback could not create code:", err)
				http.Error(w, "something went wrong", http.StatusInternalServerError)
				return
			}

			redirectURI, err := url.Parse(session.RedirectURI)
			if err != nil {
				log.Println("handler/callback could not parse redirect_uri:", err)
				http.Error(w, "something went wrong", http.StatusInternalServerError)
				return
			}

			query := redirectURI.Query()
			query.Set("code", code)
			query.Set("state", session.State)
			redirectURI.RawQuery = query.Encode()

			db.SaveLogin(w, r, session.Me)

			json.NewEncoder(w).Encode(struct {
				RedirectURI string `json:"redirectURI"`
			}{
				RedirectURI: redirectURI.String(),
			})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
