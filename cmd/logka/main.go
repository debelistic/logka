package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/debelistic/logka/internal/api"
	"github.com/debelistic/logka/internal/docker"
	"github.com/debelistic/logka/internal/server"
)

func main() {
	addr := flag.String("addr", envOr("LOGKA_ADDR", ":8080"), "HTTP listen address")
	dockerHost := flag.String("docker", envOr("DOCKER_HOST", ""), "Docker host (empty = local socket)")
	flag.Parse()

	client, err := docker.NewClient(*dockerHost)
	if err != nil {
		log.Fatalf("docker client: %v", err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		log.Printf("warning: docker not reachable yet: %v", err)
	}

	mux := http.NewServeMux()
	api.Register(mux, client)
	server.RegisterUI(mux)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           withLogging(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf("logka listening on %s\n", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
