package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/generated"
	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/k8s"
	"k8s.io/client-go/dynamic"
)

type Server struct {
	dyn dynamic.Interface
}

func NewServer(dyn dynamic.Interface) *Server {
	return &Server{dyn: dyn}
}

func ptr[T any](v T) *T { return &v }

func (s *Server) CreateInstance(w http.ResponseWriter, r *http.Request) {
	var req generated.CreateInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	instances := int64(3)
	if req.Instances != nil {
		instances = int64(*req.Instances)
	}
	storageSize := "5Gi"
	if req.StorageSize != nil {
		storageSize = *req.StorageSize
	}

	obj, err := k8s.CreateCluster(r.Context(), s.dyn, req.Name, instances, storageSize)
	if err != nil {
		http.Error(w, `{"message":"failed to create instance: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	name := obj.GetName()
	now := time.Now()
	resp := generated.Instance{
		Id:        ptr(name),
		Name:      ptr(name),
		Phase:     ptr(generated.Provisioning),
		CreatedAt: &now,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) ListInstances(w http.ResponseWriter, r *http.Request) {
	list, err := k8s.ListClusters(r.Context(), s.dyn)
	if err != nil {
		http.Error(w, `{"message":"failed to list instances"}`, http.StatusInternalServerError)
		return
	}
	out := make([]generated.Instance, 0, len(list.Items))
	for _, item := range list.Items {
		name := item.GetName()
		phase := k8s.ExtractPhase(&item)
		out = append(out, generated.Instance{
			Id:    ptr(name),
			Name:  ptr(name),
			Phase: ptr(generated.InstancePhase(phase)),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) GetInstance(w http.ResponseWriter, r *http.Request, id string) {
	obj, err := k8s.GetCluster(r.Context(), s.dyn, id)
	if err != nil {
		http.Error(w, `{"message":"instance not found"}`, http.StatusNotFound)
		return
	}
	name := obj.GetName()
	phase := k8s.ExtractPhase(obj)
	resp := generated.Instance{
		Id:    ptr(name),
		Name:  ptr(name),
		Phase: ptr(generated.InstancePhase(phase)),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) DeleteInstance(w http.ResponseWriter, r *http.Request, id string) {
	if err := k8s.DeleteCluster(r.Context(), s.dyn, id); err != nil {
		http.Error(w, `{"message":"instance not found"}`, http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) GetInstanceConnection(w http.ResponseWriter, r *http.Request, id string) {
	w.WriteHeader(http.StatusNotImplemented)
}
