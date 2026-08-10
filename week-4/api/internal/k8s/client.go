package k8s

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClient() (dynamic.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// not running in-cluster — fall back to local kubeconfig
		kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	return dynamic.NewForConfig(config)
}
