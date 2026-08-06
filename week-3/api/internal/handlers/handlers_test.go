package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/handlers"
	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/k8s"
	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/types"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

func newFakeServer(objects ...runtime.Object) (*handlers.Server, dynamic.Interface) {
	scheme := runtime.NewScheme()
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		k8s.ClusterGVR: "ClusterList",
	}, objects...)
	return handlers.NewServer(dyn), dyn
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
	srv, dyn := newFakeServer()

	body := strings.NewReader(`{"name":"api-test-1","instances":5,"storageSize":"10Gi"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/instances", body)
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
}

func TestCreateInstance_InvalidBody(t *testing.T) {
	srv, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`not json`))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateInstance_MissingInstances(t *testing.T) {
	srv, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","storageSize":"5Gi"}`))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateInstance_MissingStorageSize(t *testing.T) {
	srv, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","instances":3}`))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateInstance_Duplicate(t *testing.T) {
	srv, _ := newFakeServer(newCluster("api-test-1", ""))

	req := httptest.NewRequest(http.MethodPost, "/v1/instances", strings.NewReader(`{"name":"api-test-1","instances":3,"storageSize":"5Gi"}`))
	w := httptest.NewRecorder()

	srv.CreateInstance(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestListInstances(t *testing.T) {
	srv, _ := newFakeServer(
		newCluster("api-test-1", "Healthy"),
		newCluster("api-test-2", "Provisioning"),
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/instances", nil)
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

func TestListInstances_Empty(t *testing.T) {
	srv, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/instances", nil)
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
	srv, _ := newFakeServer(newCluster("api-test-1", "Healthy"))

	req := httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1", nil)
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

func TestGetInstance_NotFound(t *testing.T) {
	srv, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/instances/does-not-exist", nil)
	w := httptest.NewRecorder()

	srv.GetInstance(w, req, "does-not-exist")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteInstance(t *testing.T) {
	srv, _ := newFakeServer(newCluster("api-test-1", ""))

	req := httptest.NewRequest(http.MethodDelete, "/v1/instances/api-test-1", nil)
	w := httptest.NewRecorder()

	srv.DeleteInstance(w, req, "api-test-1")

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestDeleteInstance_NotFound(t *testing.T) {
	srv, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodDelete, "/v1/instances/does-not-exist", nil)
	w := httptest.NewRecorder()

	srv.DeleteInstance(w, req, "does-not-exist")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetInstanceConnection(t *testing.T) {
	srv, _ := newFakeServer(newCluster("api-test-1", "Healthy"))

	req := httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1/connection", nil)
	w := httptest.NewRecorder()

	srv.GetInstanceConnection(w, req, "api-test-1")

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotImplemented)
	}
}
