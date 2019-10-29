package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-pg/pg/v9"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
	"voice-chat-server/controller"
	"voice-chat-server/service"
)

var dbService = service.DBService{}
var sessionService = service.SessionService{
	DbService: &dbService,
}
var chatServerService = service.ChatServerService{
	DbService: &dbService,
}
var chatUserService = service.ChatUserService{
	DbService: &dbService,
}
var authController = controller.AuthController{
	UserService: &chatUserService,
	Session:     &sessionService,
}
var chatServerController = controller.ChatServerController{
	ChatServerService: &chatServerService,
}
var connectionManager = service.ChatRoomConnectionManager{
	ChatServerService: &chatServerService,
	Upgrader:          &websocket.Upgrader{},
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

var validateUrls = [...]string{"/api/server", "/api/auth/info"}

func validateTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matched := false
		for _, url := range validateUrls {
			if strings.HasPrefix(r.RequestURI, url) {
				matched = true
				break
			}
		}
		if matched {
			res := authController.ValidateToken(w, r)
			if res {
				next.ServeHTTP(w, r)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, "Token is not valid")
			}
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func doInit() {
	// init in transaction
	err := dbService.Connect()
	if err != nil {
		log.Fatal(err)
	}
	err = dbService.DB.RunInTransaction(func(tx *pg.Tx) error {
		err = sessionService.Init()
		if err != nil {
			return err
		}
		err = chatUserService.Init()
		if err != nil {
			return err
		}
		err = chatServerService.Init()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var wait time.Duration
	flag.DurationVar(&wait,
		"graceful-timeout",
		time.Second*15,
		"the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	doInit()

	log.SetFlags(0)
	r := mux.NewRouter()
	r.HandleFunc("/ws/connect", connectionManager.Connect)
	r.HandleFunc("/api/auth/login", authController.DoLogin).Methods("POST")
	r.HandleFunc("/api/auth/info", authController.GetAuthInfo).Methods("GET")
	r.HandleFunc("/api/server/list", chatServerController.ListServers).Methods("GET")
	r.HandleFunc("/api/server/info/{id}", chatServerController.GetServerInfo).Methods("GET")
	r.HandleFunc("/api/server/room", chatServerController.ListRooms).Methods("GET")
	r.Use(loggingMiddleware, validateTokenMiddleware)

	log.Printf("server start at: localhost:8080")
	srv := http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	defer dbService.CloseConnection()
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Println(err)
	}
	log.Println("server is shutting down")
	os.Exit(0)
}
