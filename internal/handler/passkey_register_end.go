package handler

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hawx.me/code/relme-auth/internal/cbor"
)

type PublicKeyCredential struct {
	AuthenticatorAttachment string                           `json:"authenticatorAttachment"`
	ID                      string                           `json:"id"`
	RawID                   string                           `json:"rawId"`
	Response                AuthenticatorAttestationResponse `json:"response"`
}

type AuthenticatorAttestationResponse struct {
	ClientDataJSON Base64[ClientData] `json:"clientDataJSON"`
	Type           string             `json:"type"`

	// webauth.create
	AttestationObject string `json:"attestationObject"`

	// webauthn.get
	AuthenticatorData string `json:"authenticatorData"`
	Signature         string `json:"signature"`
}

type ClientData struct {
	Challenge string `json:"challenge"`
	Origin    string `json:"origin"`
	Type      string `json:"type"`
}

type AttestationObject struct {
	AuthenticatorData    string `json:"authData"`
	Format               string `json:"fmt"`
	AttestationStatement string `json:"attStmt"`
}

type Base64[T any] struct {
	S string
	V T
}

func (b *Base64[T]) UnmarshalText(text []byte) error {
	buf := make([]byte, base64.RawURLEncoding.DecodedLen(len(text)))
	if _, err := base64.RawURLEncoding.Decode(buf, text); err != nil {
		return fmt.Errorf("problem decoding base64: %w", err)
	}

	var v T
	if err := json.Unmarshal(buf, &v); err != nil {
		return err
	}

	b.S = string(text)
	b.V = v
	return nil
}

func (b *Base64[T]) Bytes() []byte {
	v, _ := base64.RawURLEncoding.DecodeString(b.S)
	return v
}

type strategyClaimStore interface {
	Claim(string) (any, bool)
}

func PasskeyRegisterEnd(store strategyClaimStore, db CredentialDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "must be POST", http.StatusMethodNotAllowed)
			return
		}

		var v PublicKeyCredential
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			http.Error(w, "problem understanding what to do", http.StatusBadRequest)
			return
		}

		data := v.Response.ClientDataJSON.V

		if data.Type != "webauthn.create" || data.Origin != "http://localhost:8080" {
			http.Error(w, "how did you get here?", http.StatusBadRequest)
			return
		}

		found, ok := store.Claim(data.Challenge)
		if !ok {
			http.Error(w, "no challenge found", http.StatusBadRequest)
			return
		}
		foundID := found.(string)

		attestationData, _ := base64.RawURLEncoding.DecodeString(v.Response.AttestationObject)

		attestation, err := cbor.DecodeBytes(attestationData)
		if err != nil {
			http.Error(w, "could not check attestations", http.StatusBadRequest)
			return
		}

		authData := attestation.(map[any]any)["authData"].([]byte)

		idLen := binary.BigEndian.Uint16(authData[53:55])
		credentialID := authData[55 : 55+idLen]
		publicKeyBytes := authData[55+idLen:]

		if err := db.CreateCredential(
			foundID,
			base64.StdEncoding.EncodeToString(credentialID),
			base64.StdEncoding.EncodeToString(publicKeyBytes),
			time.Now(),
		); err != nil {
			http.Error(w, "could not create credential: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
