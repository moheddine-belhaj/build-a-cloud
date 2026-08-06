package main

import (
	"log"
	"net/http"

	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/generated"
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
	h := generated.HandlerFromMux(srv, mux)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", h))
}
