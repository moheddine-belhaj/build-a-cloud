package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ServicePort mirrors a single entry of a Service's spec.ports.
type ServicePort struct {
	Name     string
	Port     int64
	Protocol string
}

// ServiceSummary is the subset of a Service's spec/status this API surfaces
// to clients — enough to tell rw/ro/r endpoints apart and know how to reach
// them, without exposing the full raw object.
type ServiceSummary struct {
	Name       string
	Type       string
	ClusterIP  string
	ExternalIP string
	Ports      []ServicePort
}

// ListInstanceServices returns the Services CNPG manages for an instance —
// the "<name>-rw" (primary), "<name>-ro" (read-only replicas), and "<name>-r"
// (any replica) ClusterIP endpoints it creates automatically, selected by the
// "cnpg.io/cluster" label CNPG stamps on all of them. This deliberately
// excludes the "<name>-external" LoadBalancer Service this API manages
// itself (see external_service.go), which carries a different label.
func ListInstanceServices(ctx context.Context, dyn dynamic.Interface, instanceName string) (*unstructured.UnstructuredList, error) {
	return dyn.Resource(ServiceGVR).Namespace(Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "cnpg.io/cluster=" + instanceName,
	})
}

// SummarizeService extracts the fields ListInstanceServices callers need out
// of a raw Service object.
func SummarizeService(svc *unstructured.Unstructured) ServiceSummary {
	svcType, _, _ := unstructured.NestedString(svc.Object, "spec", "type")
	clusterIP, _, _ := unstructured.NestedString(svc.Object, "spec", "clusterIP")

	externalIP := ExtractLoadBalancerIP(svc)
	if externalIP == "" {
		if ips, found, _ := unstructured.NestedStringSlice(svc.Object, "spec", "externalIPs"); found && len(ips) > 0 {
			externalIP = ips[0]
		}
	}

	var ports []ServicePort
	rawPorts, _, _ := unstructured.NestedSlice(svc.Object, "spec", "ports")
	for _, rp := range rawPorts {
		p, ok := rp.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := p["name"].(string)
		protocol, _ := p["protocol"].(string)
		var port int64
		switch v := p["port"].(type) {
		case int64:
			port = v
		case float64:
			port = int64(v)
		}
		ports = append(ports, ServicePort{Name: name, Port: port, Protocol: protocol})
	}

	return ServiceSummary{
		Name:       svc.GetName(),
		Type:       svcType,
		ClusterIP:  clusterIP,
		ExternalIP: externalIP,
		Ports:      ports,
	}
}
