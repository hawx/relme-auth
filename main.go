package main

import (
	"flag"
	"fmt"

	"hawx.me/code/relme-auth/config"
	"hawx.me/code/relme-auth/handler"
	"hawx.me/code/relme-auth/state"
	"hawx.me/code/relme-auth/strategy"
	"hawx.me/code/route"
	"hawx.me/code/serve"
)

func main() {
	var (
		port           = flag.String("port", "8080", "Port to run on")
		socket         = flag.String("socket", "", "Socket to run on")
		configPath     = flag.String("config", "./config.toml", "Path to config file")
		privateKeyPath = flag.String("private-key", "./priv.pem", "Path to private key in pem format")
	)
	flag.Parse()

	conf, err := config.Read(*configPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	privateKey, err := config.ReadPrivateKey(*privateKeyPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	gitHubStrategy := strategy.GitHub(conf.GitHub.Id, conf.GitHub.Secret)

	strategies := []strategy.Strategy{
		gitHubStrategy,
	}

	authStore := state.NewStore()

	route.Handle("/", handler.Login())
	route.Handle("/authenticate", handler.Authenticate(authStore, strategies))
	route.Handle("/oauth/callback/github", handler.Callback(privateKey, authStore, gitHubStrategy))

	serve.Serve(*port, *socket, route.Default)
}
