package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/gorilla/sessions"
)

type registerPasskeyData struct {
	ID        string `json:"id"`
	Challenge string `json:"challenge"`
}

type strategySetStore interface {
	Set(key string, value any) error
}

func PasskeyRegisterStart(sessionsStore sessions.Store, strategyStore strategySetStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "must be GET", http.StatusMethodNotAllowed)
			return
		}

		session, _ := sessionsStore.Get(r, "example-session")

		meValue, ok := session.Values["me"]
		if !ok {
			http.Error(w, "not logged in", http.StatusUnauthorized)
			return
		}

		me, ok := meValue.(string)
		if !ok {
			http.Error(w, "not logged in", http.StatusUnauthorized)
			return
		}

		buf := make([]byte, 32)
		rand.Reader.Read(buf)

		data := registerPasskeyData{
			ID:        base64.StdEncoding.EncodeToString([]byte(me)),
			Challenge: base64.StdEncoding.EncodeToString(buf),
		}

		if err := strategyStore.Set(base64.RawURLEncoding.EncodeToString(buf), me); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(data)
	}
}
