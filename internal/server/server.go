package server

import (
	"io"
	"net/http"

	"github.com/gorilla/sessions"
	"hawx.me/code/mux"
	"hawx.me/code/relme-auth/internal/config"
	"hawx.me/code/relme-auth/internal/data"
	"hawx.me/code/relme-auth/internal/handler"
	"hawx.me/code/relme-auth/internal/microformats"
	"hawx.me/code/relme-auth/internal/strategy"
	"hawx.me/code/route"
)

type DB interface {
	handler.AuthDB
	handler.CallbackDB
	handler.ChooseDB
	handler.ContinueDB
	handler.ExampleDB
	handler.TokenDB
	handler.VerifyDB
	handler.WebSocketDB
}

type Templates interface {
	ExecuteTemplate(w io.Writer, tmpl string, data interface{}) error
}

func New(
	database DB,
	codeGenerator func() (string, error),
	baseURL string,
	httpClient *http.Client,
	conf config.Config,
	useTrue bool,
	webPath string,
	templates Templates,
	cookies *sessions.CookieStore,
	tokenGenerator func() (string, error),
	noRedirectClient *http.Client,
) http.Handler {
	route.Handle("/callback/continue", handler.Continue(database, codeGenerator))

	var strategies strategy.Strategies
	if useTrue {
		trueStrategy := strategy.True(baseURL)
		strategies = append(strategies, trueStrategy)

		route.Handle("/callback/true", handler.Callback(database, trueStrategy, codeGenerator))

	} else {
		pgpDatabase, _ := data.Strategy("pgp")
		pgpStrategy := strategy.PGP(pgpDatabase, baseURL, "", httpClient)
		route.Handle("/callback/pgp", handler.Callback(database, pgpStrategy, codeGenerator))
		strategies = append(strategies, pgpStrategy)

		if conf.Flickr != nil {
			flickrDatabase, _ := data.Strategy("flickr")
			flickrStrategy := strategy.Flickr(baseURL, flickrDatabase, conf.Flickr.ID, conf.Flickr.Secret, httpClient)
			route.Handle("/callback/flickr", handler.Callback(database, flickrStrategy, codeGenerator))
			strategies = append(strategies, flickrStrategy)
		}

		if conf.GitHub != nil {
			gitHubDatabase, _ := data.Strategy("github")
			gitHubStrategy := strategy.GitHub(gitHubDatabase, conf.GitHub.ID, conf.GitHub.Secret)
			route.Handle("/callback/github", handler.Callback(database, gitHubStrategy, codeGenerator))
			strategies = append(strategies, gitHubStrategy)
		}

		if conf.Twitter != nil {
			twitterDatabase, _ := data.Strategy("twitter")
			twitterStrategy := strategy.Twitter(baseURL, twitterDatabase, conf.Twitter.ID, conf.Twitter.Secret, httpClient)
			route.Handle("/callback/twitter", handler.Callback(database, twitterStrategy, codeGenerator))
			strategies = append(strategies, twitterStrategy)
		}
	}

	route.Handle("/auth", mux.Method{
		"GET":  handler.Choose(baseURL, database, strategies, templates),
		"POST": handler.Verify(database),
	})
	route.Handle("/auth/start", mux.Method{
		"GET": handler.Auth(database, strategies, httpClient),
	})

	route.Handle("/token", handler.Token(database, tokenGenerator))
	route.Handle("/pgp/authorize", handler.PGP(templates))

	route.Handle("/", handler.Example(baseURL, conf, cookies, database, templates))
	route.Handle("/redirect", handler.ExampleCallback(baseURL, cookies))
	route.Handle("/sign-out", handler.ExampleSignOut(baseURL, cookies))
	route.Handle("/revoke", handler.ExampleRevoke(baseURL, cookies, database))
	route.Handle("/privacy", handler.ExamplePrivacy(baseURL, cookies, templates))
	route.Handle("/forget", handler.ExampleForget(baseURL, cookies, database))
	route.Handle("/generate", handler.ExampleGenerate(baseURL, cookies, tokenGenerator, database, templates))

	relMe := &microformats.RelMe{Client: httpClient, NoRedirectClient: noRedirectClient}

	route.Handle("/ws", handler.WebSocket(strategies, database, relMe))
	route.Handle("/public/*path", http.StripPrefix("/public", http.FileServer(http.Dir(webPath+"/static"))))

	return route.Default
}
