package boot

import (
	"context"
	"fmt"
	"os"

	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
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

// waitForCrossplaneProvider waits until the specified crossplane provider is ready
func waitForCrossplaneProvider(dc dynamic.Interface, name string) error {
	req, err := labels.NewRequirement("pkg.crossplane.io/provider", selection.Equals, []string{name})
	if err != nil {
		return err
	}

	sel := labels.NewSelector()
	sel = sel.Add(*req)

	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "pods",
	}

	watcher, err := dc.Resource(gvr).Watch(context.Background(),
		metav1.ListOptions{LabelSelector: sel.String()},
	)
	if err != nil {
		return err
	}

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Added:
		case watch.Modified:
			obj, _ := event.Object.(*unstructured.Unstructured)

			pod := &corev1.Pod{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &pod)
			if err != nil {
				watcher.Stop()
				return err
			}

			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady &&
					cond.Status == corev1.ConditionTrue {

					watcher.Stop()
				}
			}
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

// sentinel is an object that knows how to
// start a watch on pods resources
//
// this is our implementation of `cache.Watcher`
type sentinel struct {
	client      dynamic.Interface
	timeoutSecs int64
}

// newSentinel returns a new `sentinel` object that implements `cache.Watcher`
func newSentinel(cs dynamic.Interface, timeout int64) cache.Watcher {
	return &sentinel{cs, timeout}
}

// Watch begin a watch on pods resources
func (s *sentinel) Watch(options metav1.ListOptions) (watch.Interface, error) {
	sel, err := labels.Parse("app=crossplane")
	if err != nil {
		return nil, err
	}

	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}

	return s.client.Resource(gvr).
		Namespace(kubernetes.CrossplaneSystemNamespace).
		Watch(context.Background(), metav1.ListOptions{
			LabelSelector:  sel.String(),
			TimeoutSeconds: &s.timeoutSecs,
		})
}

// just to be sure that `cache.Watcher` interface
// is being implemented by our `sentinel` struct type
var _ cache.Watcher = (*sentinel)(nil)

// waitForCrossplaneReady waits until Crossplane POD is ready
func waitForCrossplaneReady(dc dynamic.Interface) error {
	watcher := newSentinel(dc, 50)
	// create a `RetryWatcher` using initial
	// version "1" and our specialized watcher
	rw, err := toolsWatch.NewRetryWatcher("1", watcher)
	if err != nil {
		return err
	}
	defer func() {
		if x := recover(); x != nil {
			fmt.Fprintf(os.Stderr, "run time panic: %v", x)
		}
		rw.Stop()
	}()

	// process incoming event notifications
	for {
		// grab the event object
		event, ok := <-rw.ResultChan()
		if !ok {
			return fmt.Errorf("closed channel")
		}

		if et := event.Type; et != watch.Added && et != watch.Modified {
			continue
		}

		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("invalid type '%T'", event.Object)
		}
		pod := &corev1.Pod{}
		err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(obj.UnstructuredContent(), &pod)
		if err != nil {
			return err
		}

		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady &&
				cond.Status == corev1.ConditionTrue {
				return nil
			}
		}
	}
}

/*
// waitForCrossplaneReady waits until Crossplane POD is ready
func waitForCrossplaneReady2(dc dynamic.Interface) error {
	sel, err := labels.Parse("app=crossplane")
	if err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}

	watcher, err := dc.Resource(gvr).
		Namespace(kubernetes.CrossplaneSystemNamespace).
		Watch(context.Background(), metav1.ListOptions{LabelSelector: sel.String()})
	if err != nil {
		return err
	}

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Added, watch.Modified:
			obj, _ := event.Object.(*unstructured.Unstructured)
			pod := &corev1.Pod{}
			err := runtime.DefaultUnstructuredConverter.
				FromUnstructured(obj.UnstructuredContent(), &pod)
			if err != nil {
				watcher.Stop()
				return err
			}

			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady &&
					cond.Status == corev1.ConditionTrue {
					watcher.Stop()
				}
			}
		}
	}

	return nil
}
*/
