package handler

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"hawx.me/code/mux"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/microformats"
	"hawx.me/code/relme-auth/strategy"
)

const profileExpiry = 7 * 24 * time.Hour
const clientExpiry = 30 * 24 * time.Hour

type chooseStore interface {
	CreateSession(data.Session) error
	CacheClient(data.Client) error
	Client(string) (data.Client, error)
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

		client, err := getClient(clientID, redirectURI, store)
		if err != nil {
			log.Println("handler/choose failed to get client:", err)
		}

		if err := templates.ExecuteTemplate(w, "choose.gotmpl", chooseCtx{
			WebSocketURL: wsify(baseURL) + "/ws",
			ClientID:     client.ID,
			ClientName:   client.Name,
			Me:           me,
		}); err != nil {
			log.Println("handler/choose failed to write template:", err)
		}
	})
}

func wsify(s string) string {
	if strings.HasPrefix(s, "http://") {
		return "ws://" + strings.TrimPrefix(s, "http://")
	}

	return "wss://" + strings.TrimPrefix(s, "https://")
}

func getClient(clientID, redirectURI string, store chooseStore) (client data.Client, err error) {
	if foundClient, ferr := store.Client(clientID); ferr == nil {
		if foundClient.RedirectURI == redirectURI && foundClient.UpdatedAt.After(time.Now().UTC().Add(-clientExpiry)) {
			return foundClient, ferr
		}
	}

	client.ID = clientID
	client.Name = clientID
	client.UpdatedAt = time.Now().UTC()
	client.RedirectURI = redirectURI

	clientInfoResp, err := http.Get(clientID)
	if err != nil {
		return
	}
	defer clientInfoResp.Body.Close()

	if clientName, _, okerr := microformats.HApp(clientInfoResp.Body); okerr == nil {
		client.Name = clientName
	}

	err = store.CacheClient(client)
	return
}

type chooseCtx struct {
	WebSocketURL string
	ClientID     string
	ClientName   string
	Me           string
}
