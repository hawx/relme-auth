package handler

import (
	"log"
	"net/http"
	"strings"
	"time"

	"hawx.me/code/mux"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/strategy"
)

type chooseStore interface {
	CreateSession(data.Session) error
	Client(clientID, redirectURI string) (data.Client, error)
}

// Choose finds, for the "me" parameter, all authentication providers that can be
// used for authentication.
func Choose(baseURL string, store chooseStore, strategies strategy.Strategies, templates tmpl) http.Handler {
	return mux.Method{
		"GET": chooseProvider(baseURL, store, strategies, templates),
	}
}

func chooseProvider(baseURL string, store chooseStore, strategies strategy.Strategies, templates tmpl) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			me           = r.FormValue("me")
			clientID     = r.FormValue("client_id")
			redirectURI  = r.FormValue("redirect_uri")
			state        = r.FormValue("state")
			responseType = r.FormValue("response_type")
			scope        = r.FormValue("scope")
		)

		if responseType == "" {
			responseType = "id"
		}

		if me == "" || clientID == "" || redirectURI == "" || state == "" {
			http.Error(w, "missing parameter", http.StatusBadRequest)
			return
		}

		me = data.ParseProfileURL(me)
		if me == "" {
			http.Error(w, "me is invalid", http.StatusBadRequest)
			return
		}

		clientID = data.ParseClientID(clientID)
		if clientID == "" {
			http.Error(w, "client_id is invalid", http.StatusBadRequest)
			return
		}

		var scopes []string
		switch responseType {
		case "id":
			store.CreateSession(data.Session{
				Me:           me,
				ClientID:     clientID,
				RedirectURI:  redirectURI,
				State:        state,
				ResponseType: responseType,
				CreatedAt:    time.Now().UTC(),
			})

		case "code":
			scopes = strings.Fields(scope)
			if len(scopes) == 0 {
				http.Error(w, "At least one scope must be provided", http.StatusBadRequest)
				return
			}

			store.CreateSession(data.Session{
				Me:           me,
				ClientID:     clientID,
				RedirectURI:  redirectURI,
				State:        state,
				ResponseType: responseType,
				Scope:        strings.Join(scopes, " "),
				CreatedAt:    time.Now().UTC(),
			})

		default:
			http.Error(w, "Unknown response_type", http.StatusBadRequest)
			return
		}

		client, err := store.Client(clientID, redirectURI)
		if err != nil {
			log.Println("handler/choose failed to get client:", err)
			http.Error(w, "failed to retrieve client_id", http.StatusBadRequest)
			return
		}

		tmplCtx := chooseCtx{
			ClientID:   client.ID,
			ClientName: client.Name,
			Me:         me,
			Scopes:     scopes,
		}

		if err := templates.ExecuteTemplate(w, "choose.gotmpl", tmplCtx); err != nil {
			log.Println("handler/choose failed to write template:", err)
		}
	})
}

type chooseCtx struct {
	ClientID   string
	ClientName string
	Me         string
	Scopes     []string
}
