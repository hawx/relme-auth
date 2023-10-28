package handler

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"hawx.me/code/relme-auth/internal/data"
	"hawx.me/code/relme-auth/internal/microformats"
	"hawx.me/code/relme-auth/internal/strategy"

	"golang.org/x/net/websocket"
)

type WebSocketDB interface {
	Profile(string) (data.Profile, error)
	CacheProfile(data.Profile) error
}

// WebSocket returns a http.Handler that handles websocket connections. The
// client can request a set of authentication methods for a user.
func WebSocket(strategies strategy.Strategies, store WebSocketDB, relMe *microformats.RelMe) http.Handler {
	return &webSocketServer{
		strategies:  strategies,
		store:       store,
		connections: map[*conn]struct{}{},
		relMe:       relMe,
	}
}

func (s *webSocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	websocket.Handler(s.serve).ServeHTTP(w, r)
}

type webSocketServer struct {
	strategies strategy.Strategies
	store      WebSocketDB
	relMe      *microformats.RelMe

	mu          sync.RWMutex
	connections map[*conn]struct{}
}

type conn struct {
	ID  string
	Err error
	ws  *websocket.Conn
}

type profileResponse struct {
	CachedAt string
	Methods  []chooseCtxMethod
}

type eventResponse struct {
	Type   string
	Link   string
	Method chooseCtxMethod
}

func (c *conn) send(msg interface{}) {
	if err := websocket.JSON.Send(c.ws, msg); err != nil {
		log.Println("handler/websocket failed to send message:", err)
	}
}

func (s *webSocketServer) addConnection(ws *websocket.Conn) *conn {
	conn := &conn{
		ID:  "",
		Err: nil,
		ws:  ws,
	}

	s.mu.Lock()
	s.connections[conn] = struct{}{}
	s.mu.Unlock()

	return conn
}

func (s *webSocketServer) removeConnection(conn *conn) {
	s.mu.Lock()
	delete(s.connections, conn)
	s.mu.Unlock()
}

func (s *webSocketServer) serve(ws *websocket.Conn) {
	conn := s.addConnection(ws)
	defer s.removeConnection(conn)

	if err := s.serveConnection(conn); err != io.EOF {
		log.Println("handler/websocket failed to serve connection:", err)
	}
}

type profileRequest struct {
	Me          string
	ClientID    string
	RedirectURI string
	Force       bool
}

func (s *webSocketServer) serveConnection(conn *conn) error {
	for {
		var msg profileRequest
		if err := websocket.JSON.Receive(conn.ws, &msg); err != nil {
			return err
		}

		msg.Me = data.ParseProfileURL(msg.Me)

		if profile, ok := s.canUseCache(msg); ok {
			s.getFromCache(conn, msg, profile)
			continue
		}

		profile := data.Profile{
			Me:        msg.Me,
			UpdatedAt: time.Now().UTC(),
			Methods:   []data.Method{},
		}

		if err := s.readAllEvents(conn, msg, &profile); err != nil {
			log.Println("handler/websocket failed to read all events:", err)
			continue
		}
		conn.send(eventResponse{Type: "done"})

		if err := s.store.CacheProfile(profile); err != nil {
			log.Println("handler/websocket failed to cache profile:", err)
		}
	}
}

type chooseCtxMethod struct {
	Query        string
	StrategyName string
	ProfileURL   string
}

func (s *webSocketServer) canUseCache(request profileRequest) (profile data.Profile, ok bool) {
	if request.Force {
		return
	}

	profile, err := s.store.Profile(request.Me)
	if err != nil {
		return
	}

	return profile, profile.Me != "" && !profile.Expired()
}

func (s *webSocketServer) getFromCache(conn *conn, request profileRequest, profile data.Profile) {
	var methods []chooseCtxMethod
	cachedAt := profile.UpdatedAt

	for _, method := range profile.Methods {
		query := url.Values{
			"me":           {request.Me},
			"provider":     {method.Provider},
			"profile":      {method.Profile},
			"redirect_uri": {request.RedirectURI},
		}

		methods = append(methods, chooseCtxMethod{
			Query:        query.Encode(),
			StrategyName: method.Provider,
			ProfileURL:   method.Profile,
		})
	}

	conn.send(profileResponse{
		CachedAt: cachedAt.Format("2 Jan"),
		Methods:  methods,
	})
}

func (s *webSocketServer) readAllEvents(conn *conn, request profileRequest, profile *data.Profile) error {
	meCh := s.relMe.Me(request.Me, s.strategies)

	for {
		event, ok := <-meCh
		if !ok {
			return nil
		}

		switch event.Type {
		case microformats.Error:
			if event.Link == "" {
				conn.send(eventResponse{Type: "error"})
				return event.Err
			}

			conn.send(eventResponse{Type: "error", Link: event.Link})

		case microformats.PGP:
			if strategy, ok := s.strategies.IsAllowed("pgp"); ok {

				query := url.Values{
					"me":           {request.Me},
					"provider":     {strategy.Name()},
					"profile":      {event.Link},
					"redirect_uri": {request.RedirectURI},
				}

				conn.send(eventResponse{
					Type: "pgp",
					Link: event.Link,
					Method: chooseCtxMethod{
						Query:        query.Encode(),
						StrategyName: "pgp",
						ProfileURL:   event.Link,
					},
				})

				profile.Methods = append(profile.Methods, data.Method{
					Provider: strategy.Name(),
					Profile:  event.Link,
				})
			}

		case microformats.Found:
			if _, ok := s.strategies.IsAllowed(event.Link); ok {
				conn.send(eventResponse{Type: "found", Link: event.Link})
			} else {
				conn.send(eventResponse{Type: "not-supported", Link: event.Link})
			}

		case microformats.Unverified, microformats.Verified:
			if strategy, ok := s.strategies.IsAllowed(event.Link); ok {
				query := url.Values{
					"me":           {request.Me},
					"provider":     {strategy.Name()},
					"profile":      {event.Link},
					"redirect_uri": {request.RedirectURI},
				}

				typeName := "unverified"
				if event.Type == microformats.Verified {
					typeName = "verified"
				}

				conn.send(eventResponse{Type: typeName, Link: event.Link, Method: chooseCtxMethod{
					Query:        query.Encode(),
					StrategyName: strategy.Name(),
					ProfileURL:   event.Link,
				}})

				profile.Methods = append(profile.Methods, data.Method{
					Provider: strategy.Name(),
					Profile:  event.Link,
				})
			} else {
				conn.send(eventResponse{Type: "not-supported", Link: event.Link})
			}
		}
	}
}
