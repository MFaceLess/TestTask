package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"rateLimiting/pkg/config"
	"rateLimiting/pkg/db"
	"rateLimiting/pkg/handlers"
	"rateLimiting/pkg/middleware"
	"rateLimiting/pkg/token"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	cfgPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("Не удалось загрузить конфиг: %v", err)
	}

	userNameDB := os.Getenv("DB_USER")
	if userNameDB == "" {
		userNameDB = "admin"
	}
	passwordDB := os.Getenv("DB_PASSWORD")
	if passwordDB == "" {
		passwordDB = "admin"
	}
	nameDB := os.Getenv("DB_NAME")
	if nameDB == "" {
		nameDB = "db_clients_data"
	}
	hostDB := os.Getenv("DB_HOST")
	if hostDB == "" {
		hostDB = "localhost"
	}
	portDB := os.Getenv("DB_PORT")
	if portDB == "" {
		portDB = "5432"
	}

	db := db.NewDB(userNameDB, passwordDB, nameDB, hostDB, portDB)
	defer func() { db.Db.Close() }()

	rateLimiter := token.NewRateLimiter()

	userHandler := &handlers.UserHandler{
		ClientRepo: rateLimiter,
		Db:         db,
	}

	if err := db.LoadClientsFromDB(rateLimiter); err != nil {
		log.Fatalf("Ошибка при загрузке данных из таблицы %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go rateLimiter.StartRefillTicker(ctx, time.Duration(cfg.RefillInterval)*time.Second)

	r := mux.NewRouter()
	r.Use(middleware.RateLimitMiddleware(rateLimiter, cfg.BucketDefaultCapacity, cfg.DefaultRefillRate, db))
	r.Use(middleware.Panic)

	r.HandleFunc("/", userHandler.MockRequest)
	r.HandleFunc("/api/client", userHandler.AddClient).Methods(http.MethodPost)
	r.HandleFunc("/api/client/{CLIENT_ID}", userHandler.DeleteClient).Methods(http.MethodDelete)
	r.HandleFunc("/api/client/{CLIENT_ID}", userHandler.EditClient).Methods(http.MethodPut)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.ListenPort)
		log.Printf("Запуск Rate Limiting на %s ...", addr)
		if err := http.ListenAndServe(addr, r); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	log.Println("Завершение работы Rate Limiting...")
	log.Println("Rate Limiting завершил работу")
}
