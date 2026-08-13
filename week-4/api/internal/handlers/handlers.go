package handlers

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/k8s"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/middleware"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/store"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"
	"k8s.io/client-go/dynamic"
)

// postgresIdentifierRe matches a safe, unquoted Postgres identifier: starts
// with a letter/underscore, then letters/digits/underscores, up to Postgres's
// own 63-byte identifier limit. Both database and username end up as raw
// identifiers in the CNPG Cluster spec (bootstrap.initdb.database/owner), so
// this is a real trust boundary, not just cosmetic validation.
var postgresIdentifierRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)

// allowedStorageClasses is the server-side source of truth for storageClass —
// the UI only offers these as a dropdown, but that's a client-side nicety,
// not enforcement, so it's re-checked here.
var allowedStorageClasses = map[string]bool{
	"premium-perf4-stackit": true,
}

type Server struct {
	dyn       dynamic.Interface
	store     store.Store
	jwtSecret string
	jwtTTL    time.Duration
}

func NewServer(dyn dynamic.Interface, st store.Store, jwtSecret string, jwtTTL time.Duration) *Server {
	return &Server{dyn: dyn, store: st, jwtSecret: jwtSecret, jwtTTL: jwtTTL}
}

func ptr[T any](v T) *T { return &v }

// writeJSONError properly JSON-encodes the message instead of splicing it
// into a hand-written string — needed wherever the message embeds
// user-supplied input (e.g. an invalid CIDR) that could otherwise break the
// response's JSON syntax.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(types.Error{Message: ptr(message)})
}

func (s *Server) CreateInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req types.CreateInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Instances == nil {
		http.Error(w, `{"message":"instances is required"}`, http.StatusBadRequest)
		return
	}
	if req.StorageSize == nil || *req.StorageSize == "" {
		http.Error(w, `{"message":"storageSize is required"}`, http.StatusBadRequest)
		return
	}
	if req.StorageClass == nil || !allowedStorageClasses[*req.StorageClass] {
		writeJSONError(w, http.StatusBadRequest, "storageClass must be one of the offered options")
		return
	}
	if req.Database == nil || !postgresIdentifierRe.MatchString(*req.Database) {
		writeJSONError(w, http.StatusBadRequest, "database must be a valid identifier (letters, digits, underscores, starting with a letter or underscore)")
		return
	}
	if req.Username == nil || !postgresIdentifierRe.MatchString(*req.Username) {
		writeJSONError(w, http.StatusBadRequest, "username must be a valid identifier (letters, digits, underscores, starting with a letter or underscore)")
		return
	}
	for _, cidr := range req.AllowedIPs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid CIDR in allowedIPs: "+cidr)
			return
		}
	}

	// Claim the name in Postgres first: gives us a clean "already taken" error
	// and a record of ownership before ever touching Kubernetes.
	if err := s.store.RecordInstance(r.Context(), req.Name, userID); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			http.Error(w, `{"message":"an instance with this name already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"message":"failed to create instance"}`, http.StatusInternalServerError)
		return
	}

	obj, err := k8s.CreateCluster(r.Context(), s.dyn, req.Name, int64(*req.Instances), *req.StorageSize, *req.StorageClass, *req.Database, *req.Username)
	if err != nil {
		_ = s.store.DeleteInstanceRecord(r.Context(), req.Name)
		http.Error(w, `{"message":"failed to create instance: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// A dedicated LoadBalancer Service per instance — not CNPG's own "-rw"
	// Service — is what lets each instance carry its own external IP and its
	// own loadBalancerSourceRanges allowlist.
	if len(req.AllowedIPs) > 0 {
		if _, err := k8s.CreateExternalService(r.Context(), s.dyn, req.Name, req.AllowedIPs); err != nil {
			_ = k8s.DeleteCluster(r.Context(), s.dyn, req.Name)
			_ = s.store.DeleteInstanceRecord(r.Context(), req.Name)
			http.Error(w, `{"message":"failed to create instance: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	}

	name := obj.GetName()
	createdAt := obj.GetCreationTimestamp().Time
	resp := types.Instance{
		Id:             ptr(name),
		Uid:            ptr(string(obj.GetUID())),
		Name:           ptr(name),
		Phase:          ptr(types.Provisioning),
		Instances:      req.Instances,
		ReadyInstances: ptr(0),
		External:       ptr(len(req.AllowedIPs) > 0),
		StorageSize:    req.StorageSize,
		CreatedAt:      &createdAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) ListInstances(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	owned, err := s.store.ListOwned(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"message":"failed to list instances"}`, http.StatusInternalServerError)
		return
	}

	list, err := k8s.ListClusters(r.Context(), s.dyn)
	if err != nil {
		http.Error(w, `{"message":"failed to list instances"}`, http.StatusInternalServerError)
		return
	}
	out := make([]types.Instance, 0, len(owned))
	for _, item := range list.Items {
		name := item.GetName()
		if !owned[name] {
			continue
		}
		phase := k8s.ExtractPhase(&item)
		createdAt := item.GetCreationTimestamp().Time
		out = append(out, types.Instance{
			Id:             ptr(name),
			Uid:            ptr(string(item.GetUID())),
			Name:           ptr(name),
			Phase:          ptr(types.InstancePhase(phase)),
			Instances:      ptr(int(k8s.ExtractDesiredInstances(&item))),
			ReadyInstances: ptr(int(k8s.ExtractReadyInstances(&item))),
			External:       ptr(k8s.HasExternalService(r.Context(), s.dyn, name)),
			Version:        ptr(k8s.ExtractVersion(&item)),
			StorageSize:    ptr(k8s.ExtractStorageSize(&item)),
			CreatedAt:      &createdAt,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) GetInstance(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorizeOwner(w, r, id) {
		return
	}

	obj, err := k8s.GetCluster(r.Context(), s.dyn, id)
	if err != nil {
		http.Error(w, `{"message":"instance not found"}`, http.StatusNotFound)
		return
	}
	name := obj.GetName()
	phase := k8s.ExtractPhase(obj)
	createdAt := obj.GetCreationTimestamp().Time
	resp := types.Instance{
		Id:             ptr(name),
		Uid:            ptr(string(obj.GetUID())),
		Name:           ptr(name),
		Phase:          ptr(types.InstancePhase(phase)),
		Instances:      ptr(int(k8s.ExtractDesiredInstances(obj))),
		ReadyInstances: ptr(int(k8s.ExtractReadyInstances(obj))),
		External:       ptr(k8s.HasExternalService(r.Context(), s.dyn, name)),
		Version:        ptr(k8s.ExtractVersion(obj)),
		StorageSize:    ptr(k8s.ExtractStorageSize(obj)),
		CreatedAt:      &createdAt,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) DeleteInstance(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorizeOwner(w, r, id) {
		return
	}

	if err := k8s.DeleteCluster(r.Context(), s.dyn, id); err != nil {
		http.Error(w, `{"message":"instance not found"}`, http.StatusNotFound)
		return
	}
	_ = k8s.DeleteExternalService(r.Context(), s.dyn, id)
	_ = s.store.DeleteInstanceRecord(r.Context(), id)
	w.WriteHeader(http.StatusAccepted)
}

// authorizeOwner writes a 401/404 response and returns false if the caller
// isn't authenticated or doesn't own the named instance.
func (s *Server) authorizeOwner(w http.ResponseWriter, r *http.Request, name string) bool {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
		return false
	}

	ownerID, err := s.store.OwnerOf(r.Context(), name)
	if err != nil || ownerID != userID {
		http.Error(w, `{"message":"instance not found"}`, http.StatusNotFound)
		return false
	}
	return true
}

func (s *Server) GetInstanceConnection(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorizeOwner(w, r, id) {
		return
	}

	// The app-user Secret only exists once CNPG's initdb bootstrap has run —
	// using it as the readiness gate is more reliable than the Cluster's
	// status.phase, whose actual values ("Cluster in healthy state", etc.)
	// don't match the Healthy/Degraded enum this API documents.
	secret, err := k8s.GetInstanceSecret(r.Context(), s.dyn, id)
	if err != nil {
		writeJSONError(w, http.StatusConflict, "instance hasn't finished provisioning yet")
		return
	}
	username, password, dbname, err := k8s.ExtractCredentials(secret)
	if err != nil {
		writeJSONError(w, http.StatusConflict, "instance hasn't finished provisioning yet")
		return
	}

	host := id + "-rw." + k8s.Namespace + ".svc.cluster.local"
	if extSvc, err := k8s.GetExternalService(r.Context(), s.dyn, id); err == nil {
		ip := k8s.ExtractLoadBalancerIP(extSvc)
		if ip == "" {
			// allowedIPs was set at creation, but STACKIT hasn't finished
			// assigning the external IP yet — distinct from the database
			// itself not being ready.
			writeJSONError(w, http.StatusConflict, "external endpoint is still being provisioned")
			return
		}
		host = ip
	}

	resp := types.ConnectionInfo{
		Host:     ptr(host),
		Port:     ptr(5432),
		Database: ptr(dbname),
		Username: ptr(username),
		Password: ptr(password),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
