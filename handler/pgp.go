package handler

import (
	"log"
	"net/http"
)

// PGP creates a http.Handler that serves a random challenge for the user to
// clearsign.
func PGP(templates Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			clientID  = r.FormValue("client_id")
			state     = r.FormValue("state")
			challenge = r.FormValue("challenge")
		)

		if err := templates.ExecuteTemplate(w, "pgp.gotmpl", pgpCtx{
			ClientID:  clientID,
			State:     state,
			Challenge: challenge,
		}); err != nil {
			log.Println("handler/pgp failed to write template:", err)
		}
	}
}

type pgpCtx struct {
	ClientID  string
	State     string
	Challenge string
}
