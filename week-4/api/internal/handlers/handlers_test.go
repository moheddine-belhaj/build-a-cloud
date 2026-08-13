package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/handlers"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/k8s"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/middleware"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/store"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

// testUserID is the id used for the "authenticated" caller in tests, via withUser.
const testUserID int64 = 1

// fakeStore is an in-memory store.Store used so handler tests don't need a
// real Postgres connection.
type fakeStore struct {
	usersByEmail map[string]store.User
	nextUserID   int64
	instances    map[string]int64 // name -> owner id
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		usersByEmail: map[string]store.User{},
		instances:    map[string]int64{},
	}
}

func (f *fakeStore) CreateUser(ctx context.Context, email, passwordHash, firstName, lastName string) (store.User, error) {
	if _, ok := f.usersByEmail[email]; ok {
		return store.User{}, store.ErrAlreadyExists
	}
	f.nextUserID++
	u := store.User{ID: f.nextUserID, Email: email, PasswordHash: passwordHash, FirstName: firstName, LastName: lastName}
	f.usersByEmail[email] = u
	return u, nil
}

func (f *fakeStore) GetUserByEmail(ctx context.Context, email string) (store.User, error) {
	u, ok := f.usersByEmail[email]
	if !ok {
		return store.User{}, store.ErrNotFound
	}
	return u, nil
}

func (f *fakeStore) RecordInstance(ctx context.Context, name string, ownerID int64) error {
	if _, ok := f.instances[name]; ok {
		return store.ErrAlreadyExists
	}
	f.instances[name] = ownerID
	return nil
}

func (f *fakeStore) DeleteInstanceRecord(ctx context.Context, name string) error {
	delete(f.instances, name)
	return nil
}

func (f *fakeStore) OwnerOf(ctx context.Context, name string) (int64, error) {
	ownerID, ok := f.instances[name]
	if !ok {
		return 0, store.ErrNotFound
	}
	return ownerID, nil
}

func (f *fakeStore) ListOwned(ctx context.Context, ownerID int64) (map[string]bool, error) {
	owned := map[string]bool{}
	for name, owner := range f.instances {
		if owner == ownerID {
			owned[name] = true
		}
	}
	return owned, nil
}

func newFakeServer(objects ...runtime.Object) (*handlers.Server, dynamic.Interface, *fakeStore) {
	scheme := runtime.NewScheme()
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		k8s.ClusterGVR: "ClusterList",
		k8s.ServiceGVR: "ServiceList",
	}, objects...)
	st := newFakeStore()
	return handlers.NewServer(dyn, st, "test-secret", time.Hour), dyn, st
}

// withUser attaches the test caller's user id to the request, as the real
// auth middleware would after validating a bearer token.
func withUser(req *http.Request) *http.Request {
	return req.WithContext(middleware.WithUserID(req.Context(), testUserID))
}

func newCluster(name, phase string) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": k8s.Namespace,
		},
	}
	if phase != "" {
		obj["status"] = map[string]interface{}{"phase": phase}
	}
	return &unstructured.Unstructured{Object: obj}
}

func decode[T any](t *testing.T, body *bytes.Buffer) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(body).Decode(&v); err != nil {
		t.Fatalf("failed to decode response body %q: %v", body.String(), err)
	}
	return v
}

func TestCreateInstance(t *testing.T) {
	srv, dyn, _ := newFakeServer()

	body := strings.NewReader(`{"name":"api-test-1","instances":5,"storageSize":"10Gi","storageClass":"premium-perf4-stackit","database":"mydb","username":"myuser"}`)
	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances", body))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	got := decode[types.Instance](t, w.Body)
	if got.Name == nil || *got.Name != "api-test-1" {
		t.Errorf("name = %v, want api-test-1", got.Name)
	}
	if got.Phase == nil || *got.Phase != types.Provisioning {
		t.Errorf("phase = %v, want Provisioning", got.Phase)
	}
	if got.CreatedAt == nil {
		t.Error("createdAt = nil, want a timestamp")
	}

	obj, err := k8s.GetCluster(req.Context(), dyn, "api-test-1")
	if err != nil {
		t.Fatalf("expected cluster to exist in k8s after create: %v", err)
	}
	instances, _, _ := unstructured.NestedInt64(obj.Object, "spec", "instances")
	if instances != 5 {
		t.Errorf("spec.instances = %d, want 5", instances)
	}
	size, _, _ := unstructured.NestedString(obj.Object, "spec", "storage", "size")
	if size != "10Gi" {
		t.Errorf("spec.storage.size = %q, want 10Gi", size)
	}
	db, _, _ := unstructured.NestedString(obj.Object, "spec", "bootstrap", "initdb", "database")
	if db != "mydb" {
		t.Errorf("spec.bootstrap.initdb.database = %q, want mydb", db)
	}
	owner, _, _ := unstructured.NestedString(obj.Object, "spec", "bootstrap", "initdb", "owner")
	if owner != "myuser" {
		t.Errorf("spec.bootstrap.initdb.owner = %q, want myuser", owner)
	}
}

func TestCreateInstance_Unauthorized(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","instances":3,"storageSize":"5Gi"}`))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCreateInstance_InvalidBody(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`not json`)))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateInstance_MissingInstances(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","storageSize":"5Gi","storageClass":"premium-perf4-stackit","database":"mydb","username":"myuser"}`)))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateInstance_MissingStorageSize(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","instances":3,"storageClass":"premium-perf4-stackit","database":"mydb","username":"myuser"}`)))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateInstance_MissingStorageClass(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","instances":3,"storageSize":"5Gi","database":"mydb","username":"myuser"}`)))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateInstance_InvalidStorageClass(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","instances":3,"storageSize":"5Gi","storageClass":"not-an-option","database":"mydb","username":"myuser"}`)))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateInstance_InvalidDatabase(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","instances":3,"storageSize":"5Gi","storageClass":"premium-perf4-stackit","database":"1; drop table users;","username":"myuser"}`)))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateInstance_InvalidUsername(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","instances":3,"storageSize":"5Gi","storageClass":"premium-perf4-stackit","database":"mydb","username":"bad user!"}`)))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateInstance_Duplicate(t *testing.T) {
	srv, _, _ := newFakeServer(newCluster("api-test-1", ""))

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","instances":3,"storageSize":"5Gi","storageClass":"premium-perf4-stackit","database":"mydb","username":"myuser"}`)))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestListInstances(t *testing.T) {
	srv, _, st := newFakeServer(
		newCluster("api-test-1", "Healthy"),
		newCluster("api-test-2", "Provisioning"),
	)
	st.instances["api-test-1"] = testUserID
	st.instances["api-test-2"] = testUserID

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances", nil))
	w := httptest.NewRecorder()

	srv.ListInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	got := decode[[]types.Instance](t, w.Body)
	if len(got) != 2 {
		t.Fatalf("got %d instances, want 2", len(got))
	}

	byName := map[string]types.Instance{}
	for _, inst := range got {
		byName[*inst.Name] = inst
	}
	if phase := byName["api-test-1"].Phase; phase == nil || *phase != types.Healthy {
		t.Errorf("api-test-1 phase = %v, want Healthy", phase)
	}
	if phase := byName["api-test-2"].Phase; phase == nil || *phase != types.Provisioning {
		t.Errorf("api-test-2 phase = %v, want Provisioning", phase)
	}
}

func TestListInstances_OnlyOwnedByCaller(t *testing.T) {
	srv, _, st := newFakeServer(
		newCluster("api-test-1", "Healthy"),
		newCluster("someone-elses", "Healthy"),
	)
	st.instances["api-test-1"] = testUserID
	st.instances["someone-elses"] = testUserID + 1

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances", nil))
	w := httptest.NewRecorder()

	srv.ListInstances(w, req)

	got := decode[[]types.Instance](t, w.Body)
	if len(got) != 1 || got[0].Name == nil || *got[0].Name != "api-test-1" {
		t.Fatalf("got %+v, want only api-test-1", got)
	}
}

func TestListInstances_Empty(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances", nil))
	w := httptest.NewRecorder()

	srv.ListInstances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	got := decode[[]types.Instance](t, w.Body)
	if len(got) != 0 {
		t.Fatalf("got %d instances, want 0", len(got))
	}
}

func TestGetInstance(t *testing.T) {
	srv, _, st := newFakeServer(newCluster("api-test-1", "Healthy"))
	st.instances["api-test-1"] = testUserID

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1", nil))
	w := httptest.NewRecorder()

	srv.GetInstance(w, req, "api-test-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	got := decode[types.Instance](t, w.Body)
	if got.Name == nil || *got.Name != "api-test-1" {
		t.Errorf("name = %v, want api-test-1", got.Name)
	}
	if got.Phase == nil || *got.Phase != types.Healthy {
		t.Errorf("phase = %v, want Healthy", got.Phase)
	}
}

func TestGetInstance_PodCountsAndCreatedAt(t *testing.T) {
	cluster := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata": map[string]interface{}{
			"name":              "api-test-1",
			"namespace":         k8s.Namespace,
			"creationTimestamp": "2026-01-02T03:04:05Z",
		},
		"spec": map[string]interface{}{
			"instances": int64(3),
		},
		"status": map[string]interface{}{
			"phase":          "Cluster in healthy state",
			"readyInstances": int64(2),
		},
	}}
	srv, _, st := newFakeServer(cluster)
	st.instances["api-test-1"] = testUserID

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1", nil))
	w := httptest.NewRecorder()

	srv.GetInstance(w, req, "api-test-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	got := decode[types.Instance](t, w.Body)
	if got.Instances == nil || *got.Instances != 3 {
		t.Errorf("instances = %v, want 3", got.Instances)
	}
	if got.ReadyInstances == nil || *got.ReadyInstances != 2 {
		t.Errorf("readyInstances = %v, want 2", got.ReadyInstances)
	}
	if got.CreatedAt == nil || got.CreatedAt.IsZero() {
		t.Errorf("createdAt = %v, want a real timestamp", got.CreatedAt)
	}
}

func TestGetInstance_NotFound(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/does-not-exist", nil))
	w := httptest.NewRecorder()

	srv.GetInstance(w, req, "does-not-exist")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetInstance_NotOwner(t *testing.T) {
	srv, _, st := newFakeServer(newCluster("api-test-1", "Healthy"))
	st.instances["api-test-1"] = testUserID + 1 // owned by someone else

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1", nil))
	w := httptest.NewRecorder()

	srv.GetInstance(w, req, "api-test-1")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteInstance(t *testing.T) {
	srv, _, st := newFakeServer(newCluster("api-test-1", ""))
	st.instances["api-test-1"] = testUserID

	req := withUser(httptest.NewRequest(http.MethodDelete, "/v1/instances/api-test-1", nil))
	w := httptest.NewRecorder()

	srv.DeleteInstance(w, req, "api-test-1")

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestDeleteInstance_NotFound(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := withUser(httptest.NewRequest(http.MethodDelete, "/v1/instances/does-not-exist", nil))
	w := httptest.NewRecorder()

	srv.DeleteInstance(w, req, "does-not-exist")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func newAppSecret(instanceName, username, password, dbname string) *unstructured.Unstructured {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      instanceName + "-app",
			"namespace": k8s.Namespace,
		},
		"data": map[string]interface{}{
			"username": enc(username),
			"password": enc(password),
			"dbname":   enc(dbname),
		},
	}}
}

func TestGetInstanceConnection_NotReady(t *testing.T) {
	srv, _, st := newFakeServer(newCluster("api-test-1", "Provisioning"))
	st.instances["api-test-1"] = testUserID

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1/connection", nil))
	w := httptest.NewRecorder()

	srv.GetInstanceConnection(w, req, "api-test-1")

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestGetInstanceConnection_InClusterHost(t *testing.T) {
	srv, _, st := newFakeServer(
		newCluster("api-test-1", "Cluster in healthy state"),
		newAppSecret("api-test-1", "appuser", "s3cret", "appdb"),
	)
	st.instances["api-test-1"] = testUserID

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1/connection", nil))
	w := httptest.NewRecorder()

	srv.GetInstanceConnection(w, req, "api-test-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	got := decode[types.ConnectionInfo](t, w.Body)
	if got.Host == nil || *got.Host != "api-test-1-rw.cnpg-system.svc.cluster.local" {
		t.Errorf("host = %v, want in-cluster DNS name", got.Host)
	}
	if got.Username == nil || *got.Username != "appuser" {
		t.Errorf("username = %v, want appuser", got.Username)
	}
	if got.Password == nil || *got.Password != "s3cret" {
		t.Errorf("password = %v, want s3cret", got.Password)
	}
	if got.Database == nil || *got.Database != "appdb" {
		t.Errorf("database = %v, want appdb", got.Database)
	}
}

func TestGetInstanceConnection_NotOwner(t *testing.T) {
	srv, _, st := newFakeServer(
		newCluster("api-test-1", "Healthy"),
		newAppSecret("api-test-1", "appuser", "s3cret", "appdb"),
	)
	st.instances["api-test-1"] = testUserID + 1

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1/connection", nil))
	w := httptest.NewRecorder()

	srv.GetInstanceConnection(w, req, "api-test-1")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func newClusterService(clusterName, suffix string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      clusterName + suffix,
			"namespace": k8s.Namespace,
			"labels": map[string]interface{}{
				"cnpg.io/cluster": clusterName,
			},
		},
		"spec": map[string]interface{}{
			"type":      "ClusterIP",
			"clusterIP": "100.82.187.136",
			"ports": []interface{}{
				map[string]interface{}{"name": "postgres", "port": int64(5432), "protocol": "TCP"},
			},
		},
	}}
}

func TestListInstanceServices(t *testing.T) {
	srv, _, st := newFakeServer(
		newCluster("api-test-1", "Healthy"),
		newClusterService("api-test-1", "-rw"),
		newClusterService("api-test-1", "-ro"),
		newClusterService("api-test-1", "-r"),
	)
	st.instances["api-test-1"] = testUserID

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1/services", nil))
	w := httptest.NewRecorder()

	srv.ListInstanceServices(w, req, "api-test-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	got := decode[[]types.ServiceInfo](t, w.Body)
	if len(got) != 3 {
		t.Fatalf("len(services) = %d, want 3, body=%s", len(got), w.Body.String())
	}
	for _, svc := range got {
		if svc.Type == nil || *svc.Type != "ClusterIP" {
			t.Errorf("type = %v, want ClusterIP", svc.Type)
		}
		if svc.ClusterIP == nil || *svc.ClusterIP != "100.82.187.136" {
			t.Errorf("clusterIP = %v, want 100.82.187.136", svc.ClusterIP)
		}
		if len(svc.Ports) != 1 || svc.Ports[0].Port == nil || *svc.Ports[0].Port != 5432 {
			t.Errorf("ports = %+v, want one port 5432", svc.Ports)
		}
	}
}

func TestListInstanceServices_NotOwner(t *testing.T) {
	srv, _, st := newFakeServer(
		newCluster("api-test-1", "Healthy"),
		newClusterService("api-test-1", "-rw"),
	)
	st.instances["api-test-1"] = testUserID + 1

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1/services", nil))
	w := httptest.NewRecorder()

	srv.ListInstanceServices(w, req, "api-test-1")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
