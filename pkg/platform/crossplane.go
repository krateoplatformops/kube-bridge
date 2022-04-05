package platform

import (
	"context"

	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/dynamic"
)

func isCrossplaneInstalled(dc dynamic.Interface) (bool, error) {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "pods",
	}

	sel, err := labels.Parse("app=crossplane")
	if err != nil {
		return false, err
	}

	list, err := dc.Resource(gvr).
		Namespace(kubernetes.CrossplaneSystemNamespace).
		List(context.Background(), metav1.ListOptions{LabelSelector: sel.String()})
	if err != nil {
		return false, err
	}

	return len(list.Items) > 0, nil
}

func createControllerConfig(dc dynamic.Interface) error {
	gvr := schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1alpha1",
		Resource: "controllerconfigs",
	}

	obj := unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "ControllerConfig",
			"apiVersion": "pkg.crossplane.io/v1alpha1",
			"metadata": map[string]interface{}{
				"name": "krateo-controllerconfig",
			},
			"spec": map[string]interface{}{
				"securityContext":    map[string]interface{}{},
				"podSecurityContext": map[string]interface{}{},
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "100m",
						"memory": "128Mi",
					},
					"requests": map[string]interface{}{
						"cpu":    "50m",
						"memory": "64Mi",
					},
				},
			},
		},
	}

	_, err := dc.Resource(gvr).
		Create(context.Background(), &obj, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func installCrossplaneProviderHelmEventually(dc dynamic.Interface) error {
	name := "provider-helm"
	ok, err := isCrossplaneProviderAlreadyInstalled(dc, "provider-helm")
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	err = createCrossplaneProviderHelm(dc)
	if err != nil {
		return err
	}

	return waitForCrossplaneProvider(dc, name)
}

func isCrossplaneProviderAlreadyInstalled(dc dynamic.Interface, name string) (bool, error) {
	req, err := labels.NewRequirement("pkg.crossplane.io/provider", selection.Equals, []string{name})
	if err != nil {
		return false, err
	}

	sel := labels.NewSelector()
	sel = sel.Add(*req)

	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}

	list, err := dc.Resource(gvr).
		Namespace(kubernetes.CrossplaneSystemNamespace).
		List(context.Background(), metav1.ListOptions{LabelSelector: sel.String()})
	if err != nil {
		return false, err
	}

	return len(list.Items) > 0, nil
}

func createCrossplaneProviderHelm(dc dynamic.Interface) error {
	gvr := schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1",
		Resource: "providers",
	}

	obj := unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Provider",
			"apiVersion": "pkg.crossplane.io/v1",
			"metadata": map[string]interface{}{
				"name": "crossplane-provider-helm",
			},
			"spec": map[string]interface{}{
				"package": "registry.upbound.io/crossplane/provider-helm:v0.9.0",
				"controllerConfigRef": map[string]interface{}{
					"name": "krateo-controllerconfig",
				},
			},
		},
	}

	_, err := dc.Resource(gvr).
		Create(context.TODO(), &obj, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func installCrossplaneProviderKubernetesEventually(dc dynamic.Interface) error {
	name := "provider-kubernetes"
	ok, err := isCrossplaneProviderAlreadyInstalled(dc, name)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	err = createCrossplaneProviderKubernetes(dc)
	if err != nil {
		return err
	}

	return waitForCrossplaneProvider(dc, name)
}

func createCrossplaneProviderKubernetes(dc dynamic.Interface) error {
	gvr := schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1",
		Resource: "providers",
	}

	obj := unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Provider",
			"apiVersion": "pkg.crossplane.io/v1",
			"metadata": map[string]interface{}{
				"name": "crossplane-provider-kubernetes",
			},
			"spec": map[string]interface{}{
				"package": "crossplane/provider-kubernetes:main",
				"controllerConfigRef": map[string]interface{}{
					"name": "krateo-controllerconfig",
				},
			},
		},
	}

	_, err := dc.Resource(gvr).
		Create(context.TODO(), &obj, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}
