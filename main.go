package main

import (
	"flag"
	"fmt"

	"hawx.me/code/mux"
	"hawx.me/code/relme-auth/config"
	"hawx.me/code/relme-auth/data/boltdb"
	"hawx.me/code/relme-auth/handler"
	"hawx.me/code/relme-auth/strategy"
	"hawx.me/code/route"
	"hawx.me/code/serve"
)

func main() {
	var (
		port       = flag.String("port", "8080", "Port to run on")
		socket     = flag.String("socket", "", "Socket to run on")
		configPath = flag.String("config", "./config.toml", "Path to config file")
		dbPath     = flag.String("db", "./db", "Path to database")
		useTrue    = flag.Bool("true", false, "Use the fake 'true' auth provider")
	)
	flag.Parse()

	conf, err := config.Read(*configPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	database, err := boltdb.Open(*dbPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer database.Close()

	// authStore := memory.NewStore()
	var strategies strategy.Strategies

	if *useTrue {
		trueStrategy := strategy.True()
		strategies = strategy.Strategies{trueStrategy}

		route.Handle("/oauth/callback/true", handler.Callback(database, trueStrategy))

	} else {
		flickrStrategy := strategy.Flickr(database, conf.Flickr.Id, conf.Flickr.Secret)
		gitHubStrategy := strategy.GitHub(database, conf.GitHub.Id, conf.GitHub.Secret)
		twitterStrategy := strategy.Twitter(database, conf.Twitter.Id, conf.Twitter.Secret)

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
		"GET":  handler.Choose(database, database, strategies),
		"POST": handler.Verify(database),
	})
	route.Handle("/auth/start", mux.Method{
		"GET": handler.Auth(database, strategies),
	})
	route.Handle("/*rest", handler.Example())

	route.Handle("/ws", handler.WebSocket(strategies, database))

	serve.Serve(*port, *socket, route.Default)
}
