package boot

import (
	"context"

	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// createNamespace creates a namespace if not exists.
func createNamespace(dc dynamic.Interface, name string) error {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "namespaces",
	}

	obj := &unstructured.Unstructured{}
	obj.SetKind("Namespace")
	obj.SetName(name)
	obj.SetLabels(map[string]string{
		kubernetes.LabelManagedBy: kubernetes.DefaultFieldManager,
	})

	_, err := dc.Resource(gvr).
		Create(context.Background(), obj, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}
