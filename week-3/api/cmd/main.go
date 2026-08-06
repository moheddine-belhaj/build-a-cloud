package main

import (
	"log"
	"net/http"

	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/handlers"
	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/k8s"
)

func main() {
	dyn, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("failed to create k8s client: %v", err)
	}
	srv := handlers.NewServer(dyn)

	mux := http.NewServeMux()
	registerDocsRoutes(mux)

	mux.HandleFunc("GET /v1/instances", srv.ListInstances)
	mux.HandleFunc("POST /v1/instances", srv.CreateInstance)
	mux.HandleFunc("GET /v1/instances/{id}", func(w http.ResponseWriter, r *http.Request) {
		srv.GetInstance(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("DELETE /v1/instances/{id}", func(w http.ResponseWriter, r *http.Request) {
		srv.DeleteInstance(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /v1/instances/{id}/connection", func(w http.ResponseWriter, r *http.Request) {
		srv.GetInstanceConnection(w, r, r.PathValue("id"))
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
