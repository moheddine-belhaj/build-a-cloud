package k8s

import (
	"context"
	"encoding/base64"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var SecretGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "secrets",
}

// appSecretName is the Secret CNPG auto-generates for the initdb "owner"
// user (see CreateCluster's bootstrap.initdb.owner) — named "<cluster>-app".
func appSecretName(instanceName string) string {
	return instanceName + "-app"
}

func GetInstanceSecret(ctx context.Context, dyn dynamic.Interface, instanceName string) (*unstructured.Unstructured, error) {
	return dyn.Resource(SecretGVR).Namespace(Namespace).Get(ctx, appSecretName(instanceName), metav1.GetOptions{})
}

// ExtractCredentials reads the username/password/dbname keys CNPG writes
// into the generated Secret. Secret .data values are base64-encoded on the
// wire — the dynamic/unstructured client has no notion of the typed
// Secret.Data []byte field, so unlike a typed client-go Secret, decoding
// here has to be done by hand.
func ExtractCredentials(secret *unstructured.Unstructured) (username, password, dbname string, err error) {
	if username, err = decodeSecretKey(secret, "username"); err != nil {
		return "", "", "", err
	}
	if password, err = decodeSecretKey(secret, "password"); err != nil {
		return "", "", "", err
	}
	if dbname, err = decodeSecretKey(secret, "dbname"); err != nil {
		return "", "", "", err
	}
	return username, password, dbname, nil
}

func decodeSecretKey(secret *unstructured.Unstructured, key string) (string, error) {
	encoded, found, err := unstructured.NestedString(secret.Object, "data", key)
	if err != nil || !found {
		return "", fmt.Errorf("secret data key %q not found", key)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("secret data key %q is not valid base64: %w", key, err)
	}
	return string(decoded), nil
}
