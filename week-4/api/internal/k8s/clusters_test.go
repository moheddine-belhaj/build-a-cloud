package k8s

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

func newFakeClient(objects ...runtime.Object) dynamic.Interface {
	scheme := runtime.NewScheme()
	return fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		ClusterGVR: "ClusterList",
	}, objects...)
}

func newCluster(name, phase string) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": Namespace,
		},
	}
	if phase != "" {
		obj["status"] = map[string]interface{}{"phase": phase}
	}
	return &unstructured.Unstructured{Object: obj}
}

func TestCreateCluster(t *testing.T) {
	dyn := newFakeClient()

	obj, err := CreateCluster(context.Background(), dyn, "api-test-1", 3, "5Gi", "premium-perf4-stackit", "appdb", "appuser")
	if err != nil {
		t.Fatalf("CreateCluster returned error: %v", err)
	}
	if obj.GetName() != "api-test-1" {
		t.Errorf("name = %q, want %q", obj.GetName(), "api-test-1")
	}
	if obj.GetNamespace() != Namespace {
		t.Errorf("namespace = %q, want %q", obj.GetNamespace(), Namespace)
	}

	instances, found, err := unstructured.NestedInt64(obj.Object, "spec", "instances")
	if err != nil || !found || instances != 3 {
		t.Errorf("spec.instances = %v (found=%v, err=%v), want 3", instances, found, err)
	}
	size, found, err := unstructured.NestedString(obj.Object, "spec", "storage", "size")
	if err != nil || !found || size != "5Gi" {
		t.Errorf("spec.storage.size = %v (found=%v, err=%v), want 5Gi", size, found, err)
	}
	storageClass, found, err := unstructured.NestedString(obj.Object, "spec", "storage", "storageClass")
	if err != nil || !found || storageClass != "premium-perf4-stackit" {
		t.Errorf("spec.storage.storageClass = %v (found=%v, err=%v), want premium-perf4-stackit", storageClass, found, err)
	}
	db, found, err := unstructured.NestedString(obj.Object, "spec", "bootstrap", "initdb", "database")
	if err != nil || !found || db != "appdb" {
		t.Errorf("spec.bootstrap.initdb.database = %v (found=%v, err=%v), want appdb", db, found, err)
	}
	owner, found, err := unstructured.NestedString(obj.Object, "spec", "bootstrap", "initdb", "owner")
	if err != nil || !found || owner != "appuser" {
		t.Errorf("spec.bootstrap.initdb.owner = %v (found=%v, err=%v), want appuser", owner, found, err)
	}
}

func TestCreateCluster_AlreadyExists(t *testing.T) {
	dyn := newFakeClient(newCluster("api-test-1", ""))

	_, err := CreateCluster(context.Background(), dyn, "api-test-1", 3, "5Gi", "premium-perf4-stackit", "appdb", "appuser")
	if err == nil {
		t.Fatal("expected error creating a duplicate cluster, got nil")
	}
	if !apierrors.IsAlreadyExists(err) {
		t.Errorf("expected AlreadyExists error, got: %v", err)
	}
}

func TestGetCluster(t *testing.T) {
	dyn := newFakeClient(newCluster("api-test-1", "Healthy"))

	obj, err := GetCluster(context.Background(), dyn, "api-test-1")
	if err != nil {
		t.Fatalf("GetCluster returned error: %v", err)
	}
	if obj.GetName() != "api-test-1" {
		t.Errorf("name = %q, want %q", obj.GetName(), "api-test-1")
	}
}

func TestGetCluster_NotFound(t *testing.T) {
	dyn := newFakeClient()

	_, err := GetCluster(context.Background(), dyn, "does-not-exist")
	if err == nil {
		t.Fatal("expected error getting a missing cluster, got nil")
	}
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound error, got: %v", err)
	}
}

func TestListClusters(t *testing.T) {
	dyn := newFakeClient(
		newCluster("api-test-1", "Healthy"),
		newCluster("api-test-2", "Provisioning"),
	)

	list, err := ListClusters(context.Background(), dyn)
	if err != nil {
		t.Fatalf("ListClusters returned error: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(list.Items))
	}
}

func TestListClusters_Empty(t *testing.T) {
	dyn := newFakeClient()

	list, err := ListClusters(context.Background(), dyn)
	if err != nil {
		t.Fatalf("ListClusters returned error: %v", err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("got %d items, want 0", len(list.Items))
	}
}

func TestDeleteCluster(t *testing.T) {
	dyn := newFakeClient(newCluster("api-test-1", ""))

	if err := DeleteCluster(context.Background(), dyn, "api-test-1"); err != nil {
		t.Fatalf("DeleteCluster returned error: %v", err)
	}
	if _, err := GetCluster(context.Background(), dyn, "api-test-1"); !apierrors.IsNotFound(err) {
		t.Errorf("expected cluster to be gone, got err: %v", err)
	}
}

func TestDeleteCluster_NotFound(t *testing.T) {
	dyn := newFakeClient()

	err := DeleteCluster(context.Background(), dyn, "does-not-exist")
	if err == nil {
		t.Fatal("expected error deleting a missing cluster, got nil")
	}
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFound error, got: %v", err)
	}
}

func TestExtractPhase(t *testing.T) {
	tests := []struct {
		name string
		obj  *unstructured.Unstructured
		want string
	}{
		{"phase set", newCluster("c1", "Healthy"), "Healthy"},
		{"no status", newCluster("c1", ""), "Provisioning"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractPhase(tt.obj); got != tt.want {
				t.Errorf("ExtractPhase() = %q, want %q", got, tt.want)
			}
		})
	}
}
