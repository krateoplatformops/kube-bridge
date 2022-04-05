package platform

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/dynamic"
)

type providerInfo struct {
	name    string
	version string
}

func (pi *providerInfo) image() string {
	return fmt.Sprintf("registry.upbound.io/crossplane/provider-%s:%s", pi.name, pi.version)
}

func providerHelm() *providerInfo {
	return &providerInfo{
		name:    "helm",
		version: "v0.10.0",
	}
}

func providerKubernetes() *providerInfo {
	return &providerInfo{
		name:    "kubernetes",
		version: "v0.3.0",
	}
}

func installCrossplaneProviderEventually(dc dynamic.Interface, info *providerInfo) error {
	ok, err := isCrossplaneProviderAlreadyInstalled(dc, info)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	err = createCrossplaneProvider(dc, info)
	if err != nil {
		return err
	}

	return waitForCrossplaneProvider(dc, info)
}

func isCrossplaneProviderAlreadyInstalled(dc dynamic.Interface, info *providerInfo) (bool, error) {
	req, err := labels.NewRequirement("pkg.crossplane.io/provider",
		selection.Equals, []string{fmt.Sprintf("provider-%s", info.name)})
	if err != nil {
		return false, err
	}

	sel := labels.NewSelector()
	sel = sel.Add(*req)

	return podExists(dc, sel)
}

func createCrossplaneProvider(dc dynamic.Interface, info *providerInfo) error {
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
				"name": fmt.Sprintf("crossplane-provider-%s", info.name),
			},
			"spec": map[string]interface{}{
				"package": info.image(),
				"controllerConfigRef": map[string]interface{}{
					"name": "krateo-controllerconfig",
				},
			},
		},
	}

	_, err := dc.Resource(gvr).Create(context.Background(), &obj, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

// waitForCrossplaneProvider waits until the specified crossplane provider is ready
func waitForCrossplaneProvider(dc dynamic.Interface, info *providerInfo) error {
	req, err := labels.NewRequirement("pkg.crossplane.io/provider",
		selection.Equals, []string{fmt.Sprintf("provider-%s", info.name)})
	if err != nil {
		return err
	}

	sel := labels.NewSelector()
	sel = sel.Add(*req)

	stopFn := func(cond corev1.PodCondition) bool {
		return cond.Type == corev1.PodReady &&
			cond.Status == corev1.ConditionTrue
	}

	return watchForPodStatus(dc, sel, stopFn)
}
