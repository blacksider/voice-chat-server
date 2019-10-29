package main

import (
	"context"
	"fmt"
	"github.com/go-pg/pg/v9"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
	"voice-chat-server/controller"
	"voice-chat-server/logger"
	"voice-chat-server/service"
)

var dbService = service.DBService{}
var sessionService = service.SessionService{
	UserService: &chatUserService,
	DbService:   &dbService,
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
	Session:           &sessionService,
	DbService:         &dbService,
	Upgrader:          &websocket.Upgrader{},
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		logger.Logger.Debug("Current req:", r.RequestURI)
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
	logger.Init()

	// init in transaction
	err := dbService.Connect()
	if err != nil {
		logger.Logger.Fatal(err)
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
		err = connectionManager.Init()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logger.Logger.Fatal(err)
	}
}

func main() {
	doInit()

	r := mux.NewRouter()
	r.HandleFunc("/ws/connect", connectionManager.Connect)
	r.HandleFunc("/api/auth/login", authController.DoLogin).Methods("POST")
	r.HandleFunc("/api/auth/info", authController.GetAuthInfo).Methods("GET")
	r.HandleFunc("/api/server/list", chatServerController.ListServers).Methods("GET")
	r.HandleFunc("/api/server/info/{id}", chatServerController.GetServerInfo).Methods("GET")
	r.HandleFunc("/api/server/room", chatServerController.ListRooms).Methods("GET")
	r.Use(loggingMiddleware, validateTokenMiddleware)

	logger.Logger.Info("Server start at: localhost:8080")
	srv := http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Error(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)

	defer func() {
		sessionService.Close()
		dbService.CloseConnection()
		cancel()
	}()

	err := srv.Shutdown(ctx)
	if err != nil {
		logger.Logger.Error(err)
	}
	logger.Logger.Info("Server shut down")
}
