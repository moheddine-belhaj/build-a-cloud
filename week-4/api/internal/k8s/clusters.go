package k8s

import (
	"context"

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

func CreateCluster(ctx context.Context, dyn dynamic.Interface, name string, instances int64, storageSize string) (*unstructured.Unstructured, error) {
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
					"storageClass": "premium-perf4-stackit",
				},
				"bootstrap": map[string]interface{}{
					"initdb": map[string]interface{}{
						"database": "appdb",
						"owner":    "appuser",
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
