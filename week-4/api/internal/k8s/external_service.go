package k8s

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var ServiceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "services",
}

// externalServiceName is kept distinct from CNPG's own "<name>-rw" Service:
// that one is reconciled by the CNPG operator, which would fight us over
// spec.type and loadBalancerSourceRanges if we tried to edit it directly.
func externalServiceName(instanceName string) string {
	return instanceName + "-external"
}

// CreateExternalService provisions a dedicated LoadBalancer Service exposing
// an instance's primary (read/write) Postgres endpoint outside the cluster,
// restricted to allowedCIDRs via spec.loadBalancerSourceRanges.
func CreateExternalService(ctx context.Context, dyn dynamic.Interface, instanceName string, allowedCIDRs []string) (*unstructured.Unstructured, error) {
	svc := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      externalServiceName(instanceName),
				"namespace": Namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "paas-api",
					"paas.io/instance":             instanceName,
				},
			},
			"spec": map[string]interface{}{
				"type": "LoadBalancer",
				// Same selector CNPG's own "<name>-rw" Service uses, so this
				// always follows the current primary through failover.
				"selector": map[string]interface{}{
					"cnpg.io/cluster":      instanceName,
					"cnpg.io/instanceRole": "primary",
				},
				"ports": []interface{}{
					map[string]interface{}{
						"name":       "postgres",
						"port":       int64(5432),
						"targetPort": int64(5432),
						"protocol":   "TCP",
					},
				},
				"loadBalancerSourceRanges": stringsToInterfaces(allowedCIDRs),
			},
		},
	}
	return dyn.Resource(ServiceGVR).Namespace(Namespace).Create(ctx, svc, metav1.CreateOptions{})
}

func GetExternalService(ctx context.Context, dyn dynamic.Interface, instanceName string) (*unstructured.Unstructured, error) {
	return dyn.Resource(ServiceGVR).Namespace(Namespace).Get(ctx, externalServiceName(instanceName), metav1.GetOptions{})
}

// HasExternalService reports whether this instance was created with
// allowedIPs (i.e. has a dedicated LoadBalancer Service), regardless of
// whether the cloud LB has finished assigning an external IP yet.
func HasExternalService(ctx context.Context, dyn dynamic.Interface, instanceName string) bool {
	_, err := GetExternalService(ctx, dyn, instanceName)
	return err == nil
}

// DeleteExternalService is a no-op (not an error) if the instance never had
// one — only instances created with allowedIPs get a dedicated Service.
func DeleteExternalService(ctx context.Context, dyn dynamic.Interface, instanceName string) error {
	err := dyn.Resource(ServiceGVR).Namespace(Namespace).Delete(ctx, externalServiceName(instanceName), metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// ExtractLoadBalancerIP pulls the external IP assigned by the cloud
// LoadBalancer out of a Service's status. Returns "" if not yet assigned —
// STACKIT's LB provisioning is asynchronous, same as CNPG's own phase.
func ExtractLoadBalancerIP(svc *unstructured.Unstructured) string {
	ingress, found, err := unstructured.NestedSlice(svc.Object, "status", "loadBalancer", "ingress")
	if err != nil || !found || len(ingress) == 0 {
		return ""
	}
	entry, ok := ingress[0].(map[string]interface{})
	if !ok {
		return ""
	}
	ip, _ := entry["ip"].(string)
	return ip
}

func stringsToInterfaces(s []string) []interface{} {
	out := make([]interface{}, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}
