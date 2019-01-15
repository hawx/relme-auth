package handler

import (
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"hawx.me/code/relme"
	"hawx.me/code/relme-auth/data"
	"hawx.me/code/relme-auth/strategy"

	"golang.org/x/net/websocket"
)

type Conn struct {
	ID  string
	Err error
	ws  *websocket.Conn
}

type profileResponse struct {
	CachedAt string
	Methods  []chooseCtxMethod
}

func (c *Conn) send(msg profileResponse) error {
	return websocket.JSON.Send(c.ws, msg)
}

type WebSocketServer struct {
	strategies strategy.Strategies
	database   data.CacheStore

	mu          sync.RWMutex
	connections map[*Conn]struct{}
}

func WebSocket(strategies strategy.Strategies, database data.CacheStore) http.Handler {
	return &WebSocketServer{
		strategies:  strategies,
		database:    database,
		connections: map[*Conn]struct{}{},
	}
}

func (s *WebSocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	websocket.Handler(s.serve).ServeHTTP(w, r)
}

func (s *WebSocketServer) addConnection(ws *websocket.Conn) *Conn {
	conn := &Conn{
		ID:  "",
		Err: nil,
		ws:  ws,
	}

	s.mu.Lock()
	s.connections[conn] = struct{}{}
	s.mu.Unlock()

	return conn
}

func (s *WebSocketServer) removeConnection(conn *Conn) {
	s.mu.Lock()
	delete(s.connections, conn)
	s.mu.Unlock()
}

func (s *WebSocketServer) serve(ws *websocket.Conn) {
	conn := s.addConnection(ws)
	defer s.removeConnection(conn)

	if err := s.serveConnection(conn); err != io.EOF {
		log.Println(err)
	}
}

type profileRequest struct {
	Me          string
	ClientID    string
	RedirectURI string
	Force       bool
}

func (s *WebSocketServer) serveConnection(conn *Conn) error {
	// on connect
	log.Println("connected")

	for {
		var msg profileRequest
		if err := websocket.JSON.Receive(conn.ws, &msg); err != nil {
			return err
		}

		log.Println(msg)

		methods, cachedAt, err := s.getMethodsR(msg)
		if err != nil {
			log.Println(err)
		}

		conn.send(profileResponse{
			CachedAt: cachedAt.Format("2 Jan"),
			Methods:  methods,
		})
	}
}

type chooseCtxMethod struct {
	Query        template.URL
	StrategyName string
	ProfileURL   string
}

func (s *WebSocketServer) getMethodsR(request profileRequest) (methods []chooseCtxMethod, cachedAt time.Time, err error) {
	cachedAt = time.Now().UTC()

	<-time.After(5 * time.Second)

	if !request.Force {
		if profile_, err_ := s.database.GetProfile(request.Me); err_ == nil {
			if profile_.UpdatedAt.After(cachedAt.Add(-profileExpiry)) {
				log.Println("retrieved profile from cache")
				cachedAt = profile_.UpdatedAt

				for _, method := range profile_.Methods {
					query := url.Values{
						"me":           {request.Me},
						"provider":     {method.Provider},
						"profile":      {method.Profile},
						"client_id":    {request.ClientID},
						"redirect_uri": {request.RedirectURI},
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
	}

	verifiedLinks, err := relme.FindVerified(request.Me)
	if err != nil {
		err = errors.New("Something went wrong with the redirect, sorry")
		return
	}

	found, ok := s.strategies.Allowed(verifiedLinks)
	if !ok {
		err = errors.New("No rel=\"me\" links on your profile match a known provider")
		return
	}

	profile := data.Profile{
		Me:        request.Me,
		UpdatedAt: time.Now().UTC(),
		Methods:   []data.Method{},
	}

	for profileURL, strategy := range found {
		query := url.Values{
			"me":           {request.Me},
			"provider":     {strategy.Name()},
			"profile":      {profileURL},
			"client_id":    {request.ClientID},
			"redirect_uri": {request.RedirectURI},
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

	err = s.database.CacheProfile(profile)

	return
}
