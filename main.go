package main

import (
	"flag"
	"fmt"

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
		port       = flag.String("port", "8080", "Port to run on")
		socket     = flag.String("socket", "", "Socket to run on")
		baseURL    = flag.String("base-url", "http://localhost:8080", "Where this is running")
		configPath = flag.String("config", "./config.toml", "Path to config file")
		boltdbPath = flag.String("boltdb", "", "Path to database")
		useTrue    = flag.Bool("true", false, "Use the fake 'true' auth provider")
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

	var strategies strategy.Strategies
	if *useTrue {
		trueStrategy := strategy.True(*baseURL)
		strategies = strategy.Strategies{trueStrategy}

		route.Handle("/oauth/callback/true", handler.Callback(database, trueStrategy))

	} else {
		flickrStrategy := strategy.Flickr(*baseURL, database, conf.Flickr.Id, conf.Flickr.Secret)
		gitHubStrategy := strategy.GitHub(database, conf.GitHub.Id, conf.GitHub.Secret)
		twitterStrategy := strategy.Twitter(*baseURL, database, conf.Twitter.Id, conf.Twitter.Secret)

		strategies = strategy.Strategies{
			flickrStrategy,
			gitHubStrategy,
			twitterStrategy,
		}

		route.Handle("/oauth/callback/flickr", handler.Callback(database, flickrStrategy))
		route.Handle("/oauth/callback/github", handler.Callback(database, gitHubStrategy))
		route.Handle("/oauth/callback/twitter", handler.Callback(database, twitterStrategy))
	}

	route.Handle("/auth", mux.Method{
		"GET":  handler.Choose(*baseURL, database, database, strategies),
		"POST": handler.Verify(database),
	})
	route.Handle("/auth/start", mux.Method{
		"GET": handler.Auth(database, strategies),
	})
	route.Handle("/*rest", handler.Example(*baseURL))

	route.Handle("/ws", handler.WebSocket(strategies, database))

	serve.Serve(*port, *socket, route.Default)
}
