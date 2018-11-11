package handler

import (
	"html/template"
	"net/http"
	"net/url"

	"hawx.me/code/mux"
	"hawx.me/code/relme"
	"hawx.me/code/relme-auth/store"
	"hawx.me/code/relme-auth/strategy"
)

// Choose finds, for the "me" parameter, all authentication providers that can be
// used for authentication.
func Choose(authStore store.SessionStore, strategies strategy.Strategies) http.Handler {
	return mux.Method{
		"GET": chooseProvider(authStore, strategies),
	}
}

func chooseProvider(authStore store.SessionStore, strategies strategy.Strategies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		me := r.FormValue("me")

		verifiedLinks, err := relme.FindVerified(me)
		if err != nil {
			http.Error(w, "Something went wrong with the redirect, sorry", http.StatusInternalServerError)
			return
		}

		found, ok := strategies.Allowed(verifiedLinks)
		if !ok {
			http.Error(w, "No rel=\"me\" links on your profile match a known provider", http.StatusBadRequest)
			return
		}

		var methods []chooseCtxMethod
		for profileURL, strategy := range found {
			query := url.Values{
				"me":           {me},
				"provider":     {strategy.Name()},
				"profile":      {profileURL},
				"client_id":    {r.FormValue("client_id")},
				"redirect_uri": {r.FormValue("redirect_uri")},
			}

			methods = append(methods, chooseCtxMethod{
				Query:        query.Encode(),
				StrategyName: strategy.Name(),
				ProfileURL:   profileURL,
			})
		}

		chooseTmpl.Execute(w, chooseCtx{
			ClientID: r.FormValue("client_id"),
			Me:       me,
			Methods:  methods,
		})
	})
}

type chooseCtx struct {
	ClientID string
	Me       string
	Methods  []chooseCtxMethod
}

type chooseCtxMethod struct {
	Query        string
	StrategyName string
	ProfileURL   string
}

const chooseHTML = `
<!DOCTYPE html>
<html>
  <title>relme-auth</title>
  <style>

  </style>
  <body>
    <p>Authenticate using one of the methods below to sign-in to {{ .ClientID }} as {{ .Me }}
<ul>
{{ range .Methods }}
<li>
  <a href="/auth/start?{{ .Query }}">{{ .StrategyName }} - {{ .ProfileURL }}</a>
</li>
{{ end }}
</ul>
</body>
</html>
`

var chooseTmpl = template.Must(template.New("choose").Parse(chooseHTML))
