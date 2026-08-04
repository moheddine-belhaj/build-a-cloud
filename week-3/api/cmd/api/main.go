package main


import (
	"log"
	"net/http"

	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/generated"
)

type server struct{}

func (s *server) CreateInstance(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
func (s *server) ListInstances(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(http.StatusNotImplemented) }
func (s *server) GetInstance(w http.ResponseWriter, r *http.Request, id string) { w.WriteHeader(http.StatusNotImplemented) }
func (s *server) DeleteInstance(w http.ResponseWriter, r *http.Request, id string) { w.WriteHeader(http.StatusNotImplemented) }
func (s *server) GetInstanceConnection(w http.ResponseWriter, r *http.Request, id string) { w.WriteHeader(http.StatusNotImplemented) }

func main() {
	srv := &server{}
	mux := http.NewServeMux()
	h := generated.HandlerFromMux(srv, mux)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", h))
}