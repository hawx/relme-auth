package handler

import (
	"html/template"
	"log"
	"net/http"
	"strings"

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
func Choose(baseURL string, store chooseStore, strategies strategy.Strategies, templates *template.Template) http.Handler {
	return mux.Method{
		"GET": chooseProvider(baseURL, store, strategies, templates),
	}
}

func chooseProvider(baseURL string, store chooseStore, strategies strategy.Strategies, templates *template.Template) http.Handler {
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

		switch responseType {
		case "id":
			store.CreateSession(data.Session{
				Me:           me,
				ClientID:     clientID,
				RedirectURI:  redirectURI,
				State:        state,
				ResponseType: responseType,
			})

		case "code":
			scopes := strings.Fields(scope)
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
			})

		default:
			http.Error(w, "Unknown response_type", http.StatusInternalServerError)
			return
		}

		client, err := store.Client(clientID, redirectURI)
		if err != nil {
			log.Println("handler/choose failed to get client:", err)
		}

		if err := templates.ExecuteTemplate(w, "choose.gotmpl", chooseCtx{
			ClientID:   client.ID,
			ClientName: client.Name,
			Me:         me,
		}); err != nil {
			log.Println("handler/choose failed to write template:", err)
		}
	})
}

type chooseCtx struct {
	ClientID   string
	ClientName string
	Me         string
}
