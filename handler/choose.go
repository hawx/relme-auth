package handler

import (
	"html/template"
	"log"
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
				Query:        template.URL(query.Encode()),
				StrategyName: strategy.Name(),
				ProfileURL:   profileURL,
			})
		}

		if err := chooseTmpl.Execute(w, chooseCtx{
			ClientID: r.FormValue("client_id"),
			Me:       me,
			Methods:  methods,
		}); err != nil {
			log.Println(err)
		}
	})
}

type chooseCtx struct {
	ClientID string
	Me       string
	Methods  []chooseCtxMethod
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
        <h1>Sign-in to Example App</h1>
        <h2>{{ .ClientID }}</h2>
      </header>

      <p>Use one of the methods below to sign-in as <strong>{{ .Me }}</strong></p>

      <ul class="methods">
        {{ range .Methods }}
        <li><a class="btn" href="/auth/start?{{ .Query }}"><strong>{{ .StrategyName }}</strong> as {{ .ProfileURL }}</a></li>
        {{ end }}
      </ul>

      <p class="info">
        Results cached 24 October. <a href="#">Refresh</a>.
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
