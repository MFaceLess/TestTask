package main

import (
	"context"
	"flag"
	"fmt"
	"loadBalancer/pkg/backend"
	"loadBalancer/pkg/config"
	"loadBalancer/pkg/handlers"
	"loadBalancer/pkg/middleware"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	RoundRobinAlg = "round_robin"
	RandomAlg     = "random"
	LeastConnAlg  = "least_conn"
)

func main() {
	cfgPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("Не удалось загрузить конфиг: %v", err)
	}

	var strategy backend.BalancerStrategy
	switch cfg.Algorithm {
	case RoundRobinAlg:
		strategy = &backend.RoundRobinStrategy{}
	case RandomAlg:
		strategy = &backend.RandomStrategy{}
	case LeastConnAlg:
		strategy = &backend.LeastConnectionsStrategy{}
	default:
		log.Fatalf("Неизвестный алгоритм балансировки: %s", cfg.Algorithm)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := backend.NewBackendPool(cfg.Backends)
	pool.SetStrategy(strategy)
	go pool.HealthCheck(ctx, time.Duration(cfg.HealthCheckInterval)*time.Second)

	handler := handlers.SetupProxyHandler(pool)

	http.Handle("/", middleware.Panic(middleware.LoggingMiddleware(handler)))

	go func() {
		addr := fmt.Sprintf(":%d", cfg.ListenPort)
		log.Printf("Запуск Load Balancer на %s ...", addr)
		if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Завершение работы Load Balancer...")
	cancel()
	log.Println("Load Balancer завершил работу")
}
