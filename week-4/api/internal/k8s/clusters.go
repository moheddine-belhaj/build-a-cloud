package k8s

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

// helper to pull fields back out of the unstructured object safely
func ExtractPhase(u *unstructured.Unstructured) string {
	phase, found, err := unstructured.NestedString(u.Object, "status", "phase")
	if err != nil || !found {
		return "Provisioning"
	}
	return phase
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
