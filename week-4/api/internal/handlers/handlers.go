package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/k8s"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/middleware"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/store"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// postgresIdentifierRe matches a safe, unquoted Postgres identifier: starts
// with a letter/underscore, then letters/digits/underscores, up to Postgres's
// own 63-byte identifier limit. Both database and username end up as raw
// identifiers in the CNPG Cluster spec (bootstrap.initdb.database/owner), so
// this is a real trust boundary, not just cosmetic validation.
var postgresIdentifierRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)

// allowedStorageClasses is the server-side source of truth for storageClass 
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
// into a hand-written string  needed wherever the message embeds
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

	// A dedicated LoadBalancer Service per instance  not CNPG's own "-rw"
	// Service  is what lets each instance carry its own external IP and its
	// own loadBalancerSourceRanges allowlist.
	if len(req.AllowedIPs) > 0 {
		if _, err := k8s.CreateExternalService(r.Context(), s.dyn, req.Name, req.AllowedIPs); err != nil {
			_ = k8s.DeleteCluster(r.Context(), s.dyn, req.Name)
			_ = s.store.DeleteInstanceRecord(r.Context(), req.Name)
			http.Error(w, `{"message":"failed to create instance: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	}

	s.audit(r, userID, ActionInstanceCreated, &req.Name, map[string]any{
		"instances":   *req.Instances,
		"storageSize": *req.StorageSize,
		"external":    len(req.AllowedIPs) > 0,
	})

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
		out = append(out, s.instanceFromCluster(r.Context(), &item))
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
	resp := s.instanceFromCluster(r.Context(), obj)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// instanceFromCluster builds the API's Instance view straight off a live
// Cluster object  shared by every endpoint that returns full instance
// state (list, get, update) so they can't drift from each other.
func (s *Server) instanceFromCluster(ctx context.Context, obj *unstructured.Unstructured) types.Instance {
	name := obj.GetName()
	createdAt := obj.GetCreationTimestamp().Time

	external := false
	var allowedIPs []string
	if extSvc, err := k8s.GetExternalService(ctx, s.dyn, name); err == nil {
		external = true
		allowedIPs = k8s.ExtractSourceRanges(extSvc)
	}

	return types.Instance{
		Id:             ptr(name),
		Uid:            ptr(string(obj.GetUID())),
		Name:           ptr(name),
		Phase:          ptr(types.InstancePhase(k8s.ExtractPhase(obj))),
		Instances:      ptr(int(k8s.ExtractDesiredInstances(obj))),
		ReadyInstances: ptr(int(k8s.ExtractReadyInstances(obj))),
		External:       ptr(external),
		AllowedIPs:     allowedIPs,
		Version:        ptr(k8s.ExtractVersion(obj)),
		StorageSize:    ptr(k8s.ExtractStorageSize(obj)),
		CreatedAt:      &createdAt,
	}
}

// UpdateInstance applies a partial update: pod count and storage size are
// patched directly onto the Cluster CR, while allowedIPs is reconciled
// against the separate <name>-external Service. name, storageClass,
// database, and username can't be changed after creation, so they're not
// accepted here.
func (s *Server) UpdateInstance(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorizeOwner(w, r, id) {
		return
	}

	var req types.UpdateInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Instances == nil && req.StorageSize == nil && req.AllowedIPs == nil {
		writeJSONError(w, http.StatusBadRequest, "no updatable fields provided (instances, storageSize, allowedIPs)")
		return
	}
	if req.Instances != nil && (*req.Instances < 1 || *req.Instances > 5) {
		writeJSONError(w, http.StatusBadRequest, "instances must be between 1 and 5")
		return
	}
	for _, cidr := range req.AllowedIPs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid CIDR in allowedIPs: "+cidr)
			return
		}
	}

	obj, err := k8s.GetCluster(r.Context(), s.dyn, id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "instance not found")
		return
	}

	var instancesPatch *int64
	if req.Instances != nil {
		instancesPatch = ptr(int64(*req.Instances))
	}
	if req.StorageSize != nil {
		if err := k8s.ValidateStorageIncrease(k8s.ExtractStorageSize(obj), *req.StorageSize); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	if instancesPatch != nil || req.StorageSize != nil {
		obj, err = k8s.PatchClusterSpec(r.Context(), s.dyn, id, instancesPatch, req.StorageSize)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "failed to update instance: "+err.Error())
			return
		}
	}

	if req.AllowedIPs != nil {
		if err := k8s.SyncExternalService(r.Context(), s.dyn, id, req.AllowedIPs); err != nil {
			writeJSONError(w, http.StatusBadRequest, "failed to update external access: "+err.Error())
			return
		}
	}

	changed := map[string]any{}
	if req.Instances != nil {
		changed["instances"] = *req.Instances
	}
	if req.StorageSize != nil {
		changed["storageSize"] = *req.StorageSize
	}
	if req.AllowedIPs != nil {
		changed["allowedIPs"] = req.AllowedIPs
	}
	userID, _ := middleware.UserIDFromContext(r.Context())
	s.audit(r, userID, ActionInstanceUpdated, &id, changed)

	resp := s.instanceFromCluster(r.Context(), obj)
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

	// Recorded before DeleteInstanceRecord purely for clarity of intent 
	// audit_logs.instance_name has no FK to instances, so this row survives
	// the delete either way (see schema.sql).
	userID, _ := middleware.UserIDFromContext(r.Context())
	s.audit(r, userID, ActionInstanceDeleted, &id, nil)

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

// ListInstanceServices returns the Postgres endpoints CNPG exposes for an
// instance  the "-rw"/"-ro"/"-r" ClusterIP Services it manages
// automatically  so the UI can show connection targets beyond just the
// primary without the caller needing kubectl access.
func (s *Server) ListInstanceServices(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorizeOwner(w, r, id) {
		return
	}

	list, err := k8s.ListInstanceServices(r.Context(), s.dyn, id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list services")
		return
	}

	summaries := make([]k8s.ServiceSummary, 0, len(list.Items)+1)
	for i := range list.Items {
		summaries = append(summaries, k8s.SummarizeService(&list.Items[i]))
	}
	// The "<name>-external" LoadBalancer Service is managed by this API, not
	// CNPG (see external_service.go), so it carries a different label than
	// the "cnpg.io/cluster" one ListInstanceServices selects on  fetched
	// separately here so it still shows up alongside the rw/ro/r endpoints.
	if extSvc, err := k8s.GetExternalService(r.Context(), s.dyn, id); err == nil {
		summaries = append(summaries, k8s.SummarizeService(extSvc))
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Name < summaries[j].Name })

	out := make([]types.ServiceInfo, 0, len(summaries))
	for _, summary := range summaries {
		ports := make([]types.ServicePort, 0, len(summary.Ports))
		for _, p := range summary.Ports {
			ports = append(ports, types.ServicePort{
				Name:     ptr(p.Name),
				Port:     ptr(int(p.Port)),
				Protocol: ptr(p.Protocol),
			})
		}
		out = append(out, types.ServiceInfo{
			Name:       ptr(summary.Name),
			Type:       ptr(summary.Type),
			ClusterIP:  ptr(summary.ClusterIP),
			ExternalIP: ptr(summary.ExternalIP),
			Ports:      ports,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

var (
	// errInstanceNotReady and errExternalIPPending are sentinels so callers
	// of resolveInstanceConnection (GetInstanceConnection, RunInstanceQuery)
	// can each turn them into a 409 with the exact same message, without
	// duplicating the two conditions that produce them.
	errInstanceNotReady  = errors.New("instance hasn't finished provisioning yet")
	errExternalIPPending = errors.New("external endpoint is still being provisioned")
)

// resolveInstanceConnection resolves everything needed to open a connection
// to one instance's own database: host/port/credentials from the CNPG-
// managed app Secret, and  if the instance was created with allowedIPs 
// its external LoadBalancer IP in place of the in-cluster DNS name. This is
// the single place that logic lives; GetInstanceConnection and
// RunInstanceQuery both call it rather than each resolving it themselves.
func (s *Server) resolveInstanceConnection(ctx context.Context, id string) (types.ConnectionInfo, error) {
	// The app-user Secret only exists once CNPG's initdb bootstrap has run 
	// using it as the readiness gate is more reliable than the Cluster's
	// status.phase, whose actual values ("Cluster in healthy state", etc.)
	// don't match the Healthy/Degraded enum this API documents.
	secret, err := k8s.GetInstanceSecret(ctx, s.dyn, id)
	if err != nil {
		return types.ConnectionInfo{}, errInstanceNotReady
	}
	username, password, dbname, err := k8s.ExtractCredentials(secret)
	if err != nil {
		return types.ConnectionInfo{}, errInstanceNotReady
	}

	host := id + "-rw." + k8s.Namespace + ".svc.cluster.local"
	if extSvc, err := k8s.GetExternalService(ctx, s.dyn, id); err == nil {
		ip := k8s.ExtractLoadBalancerIP(extSvc)
		if ip == "" {
			// allowedIPs was set at creation, but STACKIT hasn't finished
			// assigning the external IP yet  distinct from the database
			// itself not being ready.
			return types.ConnectionInfo{}, errExternalIPPending
		}
		host = ip
	}

	return types.ConnectionInfo{
		Host:     ptr(host),
		Port:     ptr(5432),
		Database: ptr(dbname),
		Username: ptr(username),
		Password: ptr(password),
	}, nil
}

func (s *Server) GetInstanceConnection(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorizeOwner(w, r, id) {
		return
	}

	resp, err := s.resolveInstanceConnection(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}

	userID, _ := middleware.UserIDFromContext(r.Context())
	s.audit(r, userID, ActionCredentialsView, &id, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
