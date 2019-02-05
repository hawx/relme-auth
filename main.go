package main

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"hawx.me/code/mux"
	"hawx.me/code/relme-auth/config"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/data/boltdb"
	"hawx.me/code/relme-auth/data/memory"
	"hawx.me/code/relme-auth/handler"
	"hawx.me/code/relme-auth/strategy"
	"hawx.me/code/route"
	"hawx.me/code/serve"
)

func printHelp() {
	fmt.Println(`Usage: relme-auth [options]

  relme-auth is a web service for authenticating with 3rd party
  auth providers.

  The providers implemented are:
   * GitHub
   * Flickr
   * Twitter

 CONFIGURATION
   --config PATH='./config.toml'
     Configuration file to use, this defines the secrets for
     communicating with 3rd party authentication providers.

   --base-url URL='http://localhost:8080'
     Where this app is going to be accessible from.

   --example-secret SECRET
     This sets the secret used for sessions made by the example
     site. If left unset then no example site will be served.

   --true
     Use the fake 'true' authentication provider. This should
     only be used locally for testing as it says everyone is
     authenticated!

 DATA
   By default riviera runs with an in memory database.

   --boltdb PATH
      Use the boltdb file at the given path.

 SERVE
   --port PORT='8080'
      Serve on given port.

   --socket SOCK
      Serve at given socket, instead.`)
}

func main() {
	var (
		port          = flag.String("port", "8080", "Port to run on")
		socket        = flag.String("socket", "", "Socket to run on")
		baseURL       = flag.String("base-url", "http://localhost:8080", "Where this is running")
		configPath    = flag.String("config", "./config.toml", "Path to config file")
		boltdbPath    = flag.String("boltdb", "", "Path to database")
		exampleSecret = flag.String("example-secret", "", "Session secret for example site")
		useTrue       = flag.Bool("true", false, "Use the fake 'true' auth provider")
		webPath       = flag.String("web-path", "web", "Path to web/ directory")
	)
	flag.Usage = func() { printHelp() }
	flag.Parse()

	conf, err := config.Read(*configPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	var database data.Database
	if *boltdbPath != "" {
		database, err = boltdb.Open(*boltdbPath)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		database = memory.New()
	}
	defer database.Close()

	templates, err := template.ParseGlob(*webPath + "/template/*")
	if err != nil {
		fmt.Println("could not load templates:", err)
		return
	}

	var strategies strategy.Strategies
	if *useTrue {
		trueStrategy := strategy.True(*baseURL)
		strategies = append(strategies, trueStrategy)

		route.Handle("/oauth/callback/true", handler.Callback(database, trueStrategy))

	} else {
		pgpDatabase, _ := database.Strategy("pgp")
		pgpStrategy := strategy.PGP(pgpDatabase, *baseURL, "")
		route.Handle("/oauth/callback/pgp", handler.Callback(database, pgpStrategy))
		strategies = append(strategies, pgpStrategy)

		if conf.Flickr != nil {
			flickrDatabase, _ := database.Strategy("flickr")
			flickrStrategy := strategy.Flickr(*baseURL, flickrDatabase, conf.Flickr.ID, conf.Flickr.Secret)
			route.Handle("/oauth/callback/flickr", handler.Callback(database, flickrStrategy))
			strategies = append(strategies, flickrStrategy)
		}

		if conf.GitHub != nil {
			gitHubDatabase, _ := database.Strategy("github")
			gitHubStrategy := strategy.GitHub(gitHubDatabase, conf.GitHub.ID, conf.GitHub.Secret)
			route.Handle("/oauth/callback/github", handler.Callback(database, gitHubStrategy))
			strategies = append(strategies, gitHubStrategy)
		}

		if conf.Twitter != nil {
			twitterDatabase, _ := database.Strategy("twitter")
			twitterStrategy := strategy.Twitter(*baseURL, twitterDatabase, conf.Twitter.ID, conf.Twitter.Secret)
			route.Handle("/oauth/callback/twitter", handler.Callback(database, twitterStrategy))
			strategies = append(strategies, twitterStrategy)
		}
	}

	route.Handle("/auth", mux.Method{
		"GET":  handler.Choose(*baseURL, database, database, strategies, templates),
		"POST": handler.Verify(database),
	})
	route.Handle("/auth/start", mux.Method{
		"GET": handler.Auth(database, strategies),
	})

	route.Handle("/token", handler.Token(database))
	route.Handle("/pgp/authorize", handler.PGP(templates))

	if *exampleSecret != "" {
		exampleSessionStore := sessions.NewCookieStore([]byte(*exampleSecret))

		route.Handle("/", handler.Example(*baseURL, conf, exampleSessionStore, templates))
		route.Handle("/callback", handler.ExampleCallback(*baseURL, exampleSessionStore))
		route.Handle("/sign-out", handler.ExampleSignOut(*baseURL, exampleSessionStore))
	}

	route.Handle("/ws", handler.WebSocket(strategies, database))
	route.Handle("/public/*path", http.StripPrefix("/public", http.FileServer(http.Dir("web/static"))))

	serve.Serve(*port, *socket, context.ClearHandler(route.Default))
}
