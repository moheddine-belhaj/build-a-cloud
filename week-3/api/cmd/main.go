package main

import (
	"context"
	"log"
	"net/http"

	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/config"
	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/db"
	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/handlers"
	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/k8s"
	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	conn, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer conn.Close()

	dyn, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("failed to create k8s client: %v", err)
	}
	srv := handlers.NewServer(dyn, conn, cfg.JWTSecret, cfg.JWTTTL)

	mux := http.NewServeMux()
	registerDocsRoutes(mux)

	mux.HandleFunc("POST /v1/auth/register", srv.Register)
	mux.HandleFunc("POST /v1/auth/login", srv.Login)

	requireAuth := middleware.RequireAuth(cfg.JWTSecret)
	mux.HandleFunc("GET /v1/instances", requireAuth(srv.ListInstances))
	mux.HandleFunc("POST /v1/instances", requireAuth(srv.CreateInstance))
	mux.HandleFunc("GET /v1/instances/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		srv.GetInstance(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("DELETE /v1/instances/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		srv.DeleteInstance(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("GET /v1/instances/{id}/connection", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		srv.GetInstanceConnection(w, r, r.PathValue("id"))
	}))

	log.Println("listening on", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, middleware.Logging(mux)))
}
