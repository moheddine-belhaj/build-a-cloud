package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

const Namespace = "cnpg-system"

var ClusterGVR = schema.GroupVersionResource{
	Group:    "postgresql.cnpg.io",
	Version:  "v1",
	Resource: "clusters",
}

func CreateCluster(ctx context.Context, dyn dynamic.Interface, name string, instances int64, storageSize, storageClass, database, username string) (*unstructured.Unstructured, error) {
	cluster := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "postgresql.cnpg.io/v1",
			"kind":       "Cluster",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": Namespace,
			},
			"spec": map[string]interface{}{
				"instances": instances,
				"storage": map[string]interface{}{
					"size":         storageSize,
					"storageClass": storageClass,
				},
				"bootstrap": map[string]interface{}{
					"initdb": map[string]interface{}{
						"database": database,
						"owner":    username,
					},
				},
			},
		},
	}
	return dyn.Resource(ClusterGVR).Namespace(Namespace).Create(ctx, cluster, metav1.CreateOptions{})
}

func GetCluster(ctx context.Context, dyn dynamic.Interface, name string) (*unstructured.Unstructured, error) {
	return dyn.Resource(ClusterGVR).Namespace(Namespace).Get(ctx, name, metav1.GetOptions{})
}

func ListClusters(ctx context.Context, dyn dynamic.Interface) (*unstructured.UnstructuredList, error) {
	return dyn.Resource(ClusterGVR).Namespace(Namespace).List(ctx, metav1.ListOptions{})
}

func DeleteCluster(ctx context.Context, dyn dynamic.Interface, name string) error {
	return dyn.Resource(ClusterGVR).Namespace(Namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ExtractPhase reads the Cluster's live status and normalizes CNPG's raw,
// free-text phase strings (e.g. "Cluster in healthy state", "Setting up
// primary") into this API's documented Provisioning/Healthy/Degraded enum.
// The raw strings don't match that enum and were leaking to clients as-is —
// a fully healthy cluster showed as an unstyled, uncolored badge because
// nothing matched "Healthy" exactly.
func ExtractPhase(u *unstructured.Unstructured) string {
	phase, found, err := unstructured.NestedString(u.Object, "status", "phase")
	if err != nil || !found || phase == "" {
		return "Provisioning"
	}

	lower := strings.ToLower(phase)
	switch {
	case strings.Contains(lower, "healthy"):
		return "Healthy"
	case strings.Contains(lower, "unrecoverable"),
		strings.Contains(lower, "error"),
		strings.Contains(lower, "cannot determine"):
		return "Degraded"
	default:
		// Every other CNPG phase ("Setting up primary", "Creating a new
		// replica", "Upgrading cluster", "Applying configuration", ...) is a
		// transient working state.
		return "Provisioning"
	}
}

// ExtractDesiredInstances reads spec.instances — the pod count that was
// requested at creation time, available immediately even before the
// operator has reported any status.
func ExtractDesiredInstances(u *unstructured.Unstructured) int64 {
	n, found, err := unstructured.NestedInt64(u.Object, "spec", "instances")
	if err != nil || !found {
		return 0
	}
	return n
}

// ExtractReadyInstances reads status.readyInstances — 0 (not found/reported
// yet) while the cluster is still provisioning.
func ExtractReadyInstances(u *unstructured.Unstructured) int64 {
	n, found, err := unstructured.NestedInt64(u.Object, "status", "readyInstances")
	if err != nil || !found {
		return 0
	}
	return n
}

// PatchClusterSpec applies a JSON merge patch touching only spec.instances
// and/or spec.storage.size — the two Cluster fields CNPG allows changing on
// a live instance. A merge patch only overwrites the keys present, so
// patching storage.size alone leaves storageClass untouched. Either
// argument may be nil to leave that field alone.
func PatchClusterSpec(ctx context.Context, dyn dynamic.Interface, name string, instances *int64, storageSize *string) (*unstructured.Unstructured, error) {
	spec := map[string]interface{}{}
	if instances != nil {
		spec["instances"] = *instances
	}
	if storageSize != nil {
		spec["storage"] = map[string]interface{}{"size": *storageSize}
	}
	body, err := json.Marshal(map[string]interface{}{"spec": spec})
	if err != nil {
		return nil, err
	}
	return dyn.Resource(ClusterGVR).Namespace(Namespace).Patch(ctx, name, apitypes.MergePatchType, body, metav1.PatchOptions{})
}

// ValidateStorageIncrease errors out if requested is not strictly greater
// than current. CNPG's own admission webhook already rejects a shrink at
// the API server, but checking here first gives a clean 400 with a message
// that names the current size, instead of surfacing the webhook's raw
// rejection.
func ValidateStorageIncrease(current, requested string) error {
	currentQty, err := resource.ParseQuantity(current)
	if err != nil {
		return fmt.Errorf("current storage size %q could not be parsed: %w", current, err)
	}
	requestedQty, err := resource.ParseQuantity(requested)
	if err != nil {
		return fmt.Errorf("invalid storageSize %q: %w", requested, err)
	}
	if requestedQty.Cmp(currentQty) <= 0 {
		return fmt.Errorf("storageSize can only be increased, not decreased or kept the same (currently %s)", current)
	}
	return nil
}

// ExtractStorageSize reads spec.storage.size, e.g. "5Gi" — the value
// requested at creation time, set immediately, no operator status needed.
func ExtractStorageSize(u *unstructured.Unstructured) string {
	size, found, err := unstructured.NestedString(u.Object, "spec", "storage", "size")
	if err != nil || !found {
		return ""
	}
	return size
}

// ExtractVersion reads the Postgres version tag off the operand image the
// operator is actually running (status.image), falling back to an
// explicit spec.imageName override. Empty until the operator has resolved
// and reported an image, which isn't yet the case on a freshly created
// Cluster.
func ExtractVersion(u *unstructured.Unstructured) string {
	image, found, err := unstructured.NestedString(u.Object, "status", "image")
	if err != nil || !found || image == "" {
		image, found, err = unstructured.NestedString(u.Object, "spec", "imageName")
		if err != nil || !found {
			return ""
		}
	}

	// image is a full ref like "ghcr.io/cloudnative-pg/postgresql:16.4" —
	// split off the registry path first so a registry:port doesn't get
	// mistaken for the tag separator.
	segments := strings.Split(image, "/")
	last := segments[len(segments)-1]
	parts := strings.SplitN(last, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}
