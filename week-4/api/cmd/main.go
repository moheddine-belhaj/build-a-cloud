package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/config"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/db"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/handlers"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/k8s"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	mux.HandleFunc("PATCH /v1/instances/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		srv.UpdateInstance(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("DELETE /v1/instances/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		srv.DeleteInstance(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("GET /v1/instances/{id}/connection", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		srv.GetInstanceConnection(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("GET /v1/instances/{id}/services", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		srv.ListInstanceServices(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("POST /v1/instances/{id}/query", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		srv.RunInstanceQuery(w, r, r.PathValue("id"))
	}))
	mux.HandleFunc("GET /v1/audit-logs", requireAuth(srv.ListAuditLogs))
	mux.HandleFunc("GET /v1/instances/{id}/audit-logs", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		srv.ListInstanceAuditLogs(w, r, r.PathValue("id"))
	}))

	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", promhttp.Handler())
	metricsServer := &http.Server{Addr: cfg.MetricsAddr, Handler: metricsMux}

	apiServer := &http.Server{
		Addr:    cfg.Addr,
		Handler: middleware.Logging(middleware.Metrics(middleware.CORS(mux))),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Println("metrics listening on", cfg.MetricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("metrics server failed: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		log.Println("listening on", cfg.Addr)
		if err := apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("api server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received, draining in-flight requests")

	// terminationGracePeriodSeconds is 30s; leave a margin so the process
	// still exits (closing the DB pool via defer) before Kubernetes SIGKILLs it.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("api server shutdown error: %v", err)
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("metrics server shutdown error: %v", err)
	}

	wg.Wait()
	log.Println("shutdown complete")
}
