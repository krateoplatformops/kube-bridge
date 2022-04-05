package platform

import (
	"context"
	"fmt"
	"strings"

	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func createRoleBindingForCrossplaneProviders(dc dynamic.Interface) error {
	accept := func(name string) string {
		wants := []string{"provider-helm", "provider-kubernetes"}

		for _, el := range wants {
			if strings.Contains(name, el) {
				return el
			}
		}

		return ""
	}

	gvr := schema.GroupVersionResource{Version: "v1", Resource: "serviceaccounts"}

	list, err := dc.Resource(gvr).Namespace(kubernetes.CrossplaneSystemNamespace).
		List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, sa := range list.Items {
		provider := accept(sa.GetName())
		if len(provider) > 0 {
			err := createRoleBinding(dc, sa.GetName(), provider)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func createRoleBinding(dc dynamic.Interface, serviceAccount, provider string) error {
	gvr := schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "clusterrolebindings",
	}

	crb := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kubernetes.LabelManagedBy: kubernetes.DefaultFieldManager,
			},
			Name: fmt.Sprintf("%s-admin-binding", provider),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: kubernetes.CrossplaneSystemNamespace,
			},
		},
	}

	dat, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&crb)
	if err != nil {
		return err
	}

	obj := unstructured.Unstructured{}
	obj.SetUnstructuredContent(dat)

	_, err = dc.Resource(gvr).Create(context.TODO(), &obj, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}
