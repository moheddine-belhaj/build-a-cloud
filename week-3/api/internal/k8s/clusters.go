package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const namespace = "cnpg-system"

var clusterGVR = schema.GroupVersionResource{
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
				"namespace": namespace,
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
	return dyn.Resource(clusterGVR).Namespace(namespace).Create(ctx, cluster, metav1.CreateOptions{})
}

func GetCluster(ctx context.Context, dyn dynamic.Interface, name string) (*unstructured.Unstructured, error) {
	return dyn.Resource(clusterGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func ListClusters(ctx context.Context, dyn dynamic.Interface) (*unstructured.UnstructuredList, error) {
	return dyn.Resource(clusterGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

func DeleteCluster(ctx context.Context, dyn dynamic.Interface, name string) error {
	return dyn.Resource(clusterGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// helper to pull fields back out of the unstructured object safely
func ExtractPhase(u *unstructured.Unstructured) string {
	phase, found, err := unstructured.NestedString(u.Object, "status", "phase")
	if err != nil || !found {
		return "Provisioning"
	}
	return phase
}
