package handler

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"hawx.me/code/mux"
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

		client, err := getClient(clientID, redirectURI, database)
		if err != nil {
			log.Println("error getting client info:", err)
		}

		if err := chooseTmpl.Execute(w, chooseCtx{
			ClientID:   client.ID,
			ClientName: client.Name,
			Me:         me,
		}); err != nil {
			log.Println(err)
		}
	})
}

func getClient(clientID, redirectURI string, database data.Database) (client data.Client, err error) {
	if client_, err_ := database.GetClient(clientID); err_ == nil {
		if client_.RedirectURI == redirectURI && client_.UpdatedAt.After(time.Now().UTC().Add(-clientExpiry)) {
			log.Println("retrieved client from cache")
			return client_, err_
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

      .info.loading {
        display: none;
      }

      /* https://projects.lukehaas.me/css-loaders/ */
      .loader,
      .loader:before,
      .loader:after {
        border-radius: 50%;
        width: 2.5em;
        height: 2.5em;
        -webkit-animation-fill-mode: both;
        animation-fill-mode: both;
        -webkit-animation: load7 1.8s infinite ease-in-out;
        animation: load7 1.8s infinite ease-in-out;
      }
      .loader {
        color: #666666;
        font-size: 10px;
        margin: 80px auto;
        position: relative;
        text-indent: -9999em;
        -webkit-transform: translateZ(0);
        -ms-transform: translateZ(0);
        transform: translateZ(0);
        -webkit-animation-delay: -0.16s;
        animation-delay: -0.16s;
      }
      .loader:before,
      .loader:after {
        content: '';
        position: absolute;
        top: 0;
      }
      .loader:before {
        left: -3.5em;
        -webkit-animation-delay: -0.32s;
        animation-delay: -0.32s;
      }
      .loader:after {
        left: 3.5em;
      }
      @-webkit-keyframes load7 {
        0%,
        80%,
        100% {
          box-shadow: 0 2.5em 0 -1.3em;
        }
        40% {
          box-shadow: 0 2.5em 0 0;
        }
      }
      @keyframes load7 {
        0%,
        80%,
        100% {
          box-shadow: 0 2.5em 0 -1.3em;
        }
        40% {
          box-shadow: 0 2.5em 0 0;
        }
      }
      .loader.hide {
        display: none;
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

      <div class="loader"></div>
      <ul class="methods"></ul>

      <p class="info loading">
        Results cached <span class="cachedAt"></span>. <a id="refresh">Refresh</a>.
      </p>

      <footer>
        This is <a href="https://github.com/hawx/relme-auth">relme-auth</a>, an app that allows you to sign-in to websites by delegating to authentication providers using <code>rel=me</code> links
        on your homepage and other sites. <a href="https://indieauth.com/setup">Learn more</a>.
      </footer>
    </div>

    <div class="fill"></div>

    <script>
      const urlParams = new URLSearchParams(window.location.search);

      const methods = document.querySelector('.methods');
      const info = document.querySelector('.info');
      const cachedAt = document.querySelector('.cachedAt');
      const refresh = document.getElementById('refresh');
      const loader = document.querySelector('.loader');

      var socket = new WebSocket("ws://localhost:8080/ws");
      socket.onopen = function (event) {
        socket.send(JSON.stringify({
          me: urlParams.get('me'),
          clientID: urlParams.get('client_id'),
          redirectURI: urlParams.get('redirect_uri'),
          force: false,
        }));
      };

      refresh.onclick = function() {
        while (methods.firstChild) {
          methods.removeChild(methods.firstChild);
        }
        info.classList.add('loading');
        loader.classList.remove('hide');

        socket.send(JSON.stringify({
          me: urlParams.get('me'),
          clientID: urlParams.get('client_id'),
          redirectURI: urlParams.get('redirect_uri'),
          force: true,
        }));
      };

      socket.onmessage = function (event) {
        const profile = JSON.parse(event.data);
        loader.classList.add('hide');

        cachedAt.textContent = profile.CachedAt;
        info.classList.remove('loading');

        for (const method of profile.Methods) {
          const li = document.createElement('li');

          const btn = document.createElement('a');
          btn.classList.add('btn');
          btn.href = '/auth/start?' + method.Query;

          const name = document.createElement('strong');
          name.textContent = method.StrategyName;

          const asText = document.createTextNode(' as ' + method.ProfileURL);

          btn.appendChild(name);
          btn.appendChild(asText);
          li.appendChild(btn);
          methods.appendChild(li);
        }
      }
    </script>
  </body>
</html>
`

var chooseTmpl = template.Must(template.New("choose").Parse(chooseHTML))
