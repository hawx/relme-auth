package handler

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	"hawx.me/code/mux"
	"hawx.me/code/relme"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/microformats"
	"hawx.me/code/relme-auth/store"
	"hawx.me/code/relme-auth/strategy"
)

const profileExpiry = 7 * 24 * time.Hour
const clientExpiry = 30 * 24 * time.Hour

// Choose finds, for the "me" parameter, all authentication providers that can be
// used for authentication.
func Choose(authStore store.SessionStore, database data.Database, strategies strategy.Strategies) http.Handler {
	return mux.Method{
		"GET": chooseProvider(authStore, database, strategies),
	}
}

func chooseProvider(authStore store.SessionStore, database data.Database, strategies strategy.Strategies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			me          = r.FormValue("me")
			clientID    = r.FormValue("client_id")
			redirectURI = r.FormValue("redirect_uri")
		)

		methods, cachedAt, err := getMethods(me, clientID, redirectURI, strategies, database)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client, err := getClient(clientID, database)
		if err != nil {
			log.Println("error getting client info:", err)
		}

		if err := chooseTmpl.Execute(w, chooseCtx{
			ClientID:   client.ID,
			ClientName: client.Name,
			Me:         me,
			CachedAt:   cachedAt.Format("2 Jan"),
			Methods:    methods,
		}); err != nil {
			log.Println(err)
		}
	})
}

func getMethods(me string, clientID string, redirectURI string, strategies strategy.Strategies, database data.Database) (methods []chooseCtxMethod, cachedAt time.Time, err error) {
	cachedAt = time.Now().UTC()

	if profile_, err_ := database.GetProfile(me); err_ == nil {
		if profile_.UpdatedAt.After(cachedAt.Add(-profileExpiry)) {
			log.Println("retrieved profile from cache")
			cachedAt = profile_.UpdatedAt

			for _, method := range profile_.Methods {
				query := url.Values{
					"me":           {me},
					"provider":     {method.Provider},
					"profile":      {method.Profile},
					"client_id":    {clientID},
					"redirect_uri": {redirectURI},
				}

				methods = append(methods, chooseCtxMethod{
					Query:        template.URL(query.Encode()),
					StrategyName: method.Provider,
					ProfileURL:   method.Profile,
				})
			}

			return
		}
	}

	verifiedLinks, err := relme.FindVerified(me)
	if err != nil {
		err = errors.New("Something went wrong with the redirect, sorry")
		return
	}

	found, ok := strategies.Allowed(verifiedLinks)
	if !ok {
		err = errors.New("No rel=\"me\" links on your profile match a known provider")
		return
	}

	profile := data.Profile{
		Me:        me,
		UpdatedAt: time.Now().UTC(),
		Methods:   []data.Method{},
	}

	for profileURL, strategy := range found {
		query := url.Values{
			"me":           {me},
			"provider":     {strategy.Name()},
			"profile":      {profileURL},
			"client_id":    {clientID},
			"redirect_uri": {redirectURI},
		}

		methods = append(methods, chooseCtxMethod{
			Query:        template.URL(query.Encode()),
			StrategyName: strategy.Name(),
			ProfileURL:   profileURL,
		})

		profile.Methods = append(profile.Methods, data.Method{
			Provider: strategy.Name(),
			Profile:  profileURL,
		})
	}

	err = database.CacheProfile(profile)

	return
}

func getClient(clientID string, database data.Database) (client data.Client, err error) {
	if client_, err_ := database.GetClient(clientID); err_ == nil {
		if client_.UpdatedAt.After(time.Now().UTC().Add(-clientExpiry)) {
			log.Println("retrieved client from cache")
			return client_, err_
		}
	}

	client.ID = clientID
	client.Name = clientID
	client.UpdatedAt = time.Now().UTC()

	clientInfoResp, err := http.Get(clientID)
	if err != nil {
		return
	}
	defer clientInfoResp.Body.Close()

	if clientName, _, err_ := microformats.HApp(clientInfoResp.Body); err_ == nil {
		client.Name = clientName
	}

	err = database.CacheClient(client)
	return
}

type chooseCtx struct {
	ClientID   string
	ClientName string
	Me         string
	CachedAt   string
	Methods    []chooseCtxMethod
}

type chooseCtxMethod struct {
	Query        template.URL
	StrategyName string
	ProfileURL   string
}

const chooseHTML = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>relme-auth</title>
    <style>
      /* http://meyerweb.com/eric/tools/css/reset/
         v2.0 | 20110126
         License: none (public domain)
       */

      html, body, div, span, applet, object, iframe,
      h1, h2, h3, h4, h5, h6, p, blockquote, pre,
      a, abbr, acronym, address, big, cite, code,
      del, dfn, em, img, ins, kbd, q, s, samp,
      small, strike, strong, sub, sup, tt, var,
      b, u, i, center,
      dl, dt, dd, ol, ul, li,
      fieldset, form, label, legend,
      table, caption, tbody, tfoot, thead, tr, th, td,
      article, aside, canvas, details, embed,
      figure, figcaption, footer, header, hgroup,
      menu, nav, output, ruby, section, summary,
      time, mark, audio, video {
	      margin: 0;
	      padding: 0;
	      border: 0;
	      font-size: 100%;
	      font: inherit;
	      vertical-align: baseline;
      }
      /* HTML5 display-role reset for older browsers */
      article, aside, details, figcaption, figure,
      footer, header, hgroup, menu, nav, section {
	      display: block;
      }
      body {
	      line-height: 1;
      }
      ol, ul {
	      list-style: none;
      }
      blockquote, q {
	      quotes: none;
      }
      blockquote:before, blockquote:after,
      q:before, q:after {
	      content: '';
	      content: none;
      }
      table {
	      border-collapse: collapse;
	      border-spacing: 0;
      }

      html, body {
        height: 100%;
      }

      body {
        font: 1em/1.3 Verdana, Geneva, sans-serif;
        display: flex;
        flex-direction: column;
      }

      .container {
        max-width: 35rem;
        margin: 0 auto 4rem;
      }

      header {
        border-bottom: 1px solid #ccc;
        margin: 1.3rem 0;
        padding: 1.3rem 0;
      }

      h1 {
        font-size: 1.5rem;
        font-weight: bold;
      }

      h2 {
        font-size: 1.2rem;
        color: #666;
      }

      ul.methods {
        padding-left: 1rem;
      }

      ul.methods li {
        margin: 1.3rem 0;
      }

      strong {
        font-weight: bold;
      }

      a {
        color: rgb(54, 93, 169);
        text-decoration: none;
        border-bottom: 1px solid rgb(54, 93, 169);
      }

      a:hover {
        color: rgb(42, 100, 151);
        border-color: rgb(42, 100, 151);
      }

      a.btn {
        border: 1px solid;
        padding: .3rem .5rem;
        border-radius: .2rem;
        background: rgba(54, 93, 169, .1);
        border-color: rgba(54, 93, 169, .2);
      }

      a.btn:hover {
        background: rgba(42, 100, 151, .1);
        border-color: rgba(42, 100, 151, .5);
      }

      footer {
        margin: 2.6rem 0;
        font-size: .7rem;
        color: #666;
      }

      footer a {
        color: #666;
        border: none;
        text-decoration: underline;
      }

      footer a:hover {
        color: black;
      }

      .fill {
        flex: 1;
      }

      .container + .fill {
        flex: 3;
      }

      .info {
        font-style: italic;
        margin: 2.6rem 0;
      }
    </style>
  </head>
  <body>
    <div class="fill"></div>

    <div class="container">
      <header>
        <h1>Sign-in to {{ .ClientName }}</h1>
        <h2>{{ .ClientID }}</h2>
      </header>

      <p>Use one of the methods below to sign-in as <strong>{{ .Me }}</strong></p>

      <ul class="methods">
        {{ range .Methods }}
        <li><a class="btn" href="/auth/start?{{ .Query }}"><strong>{{ .StrategyName }}</strong> as {{ .ProfileURL }}</a></li>
        {{ end }}
      </ul>

      <p class="info">
        Results cached {{ .CachedAt }}. <a href="#">Refresh</a>.
      </p>

      <footer>
        This is <a href="https://github.com/hawx/relme-auth">relme-auth</a>, an app that allows you to sign-in to websites by delegating to authentication providers using <code>rel=me</code> links
        on your homepage and other sites. <a href="https://indieauth.com/setup">Learn more</a>.
      </footer>
    </div>

    <div class="fill"></div>
  </body>
</html>
`

var chooseTmpl = template.Must(template.New("choose").Parse(chooseHTML))
