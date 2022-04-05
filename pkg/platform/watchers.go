package platform

import (
	"context"
	"fmt"
	"os"

	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
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

// waitForCrossplaneProvider waits until the specified crossplane provider is ready
func waitForCrossplaneProvider(dc dynamic.Interface, name string) error {
	req, err := labels.NewRequirement("pkg.crossplane.io/provider", selection.Equals, []string{name})
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

// waitForCrossplaneReady waits until Crossplane PODs are ready
func waitForCrossplaneReady(dc dynamic.Interface) error {
	sel, err := labels.Parse("app=crossplane")
	if err != nil {
		return err
	}

	stopFn := func(cond corev1.PodCondition) bool {
		return cond.Type == corev1.PodReady &&
			cond.Status == corev1.ConditionTrue
	}

	return watchForPodStatus(dc, sel, stopFn)
}

func watchForPodStatus(dc dynamic.Interface, sel labels.Selector, stopFn func(corev1.PodCondition) bool) error {
	watchFn := func(_ metav1.ListOptions) (watch.Interface, error) {
		timeoutSecs := int64(120)

		gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}

		return dc.Resource(gvr).
			Namespace(kubernetes.CrossplaneSystemNamespace).
			Watch(context.Background(), metav1.ListOptions{
				LabelSelector:  sel.String(),
				TimeoutSeconds: &timeoutSecs,
			})
	}

	// create a `RetryWatcher` using initial version "1" and our specialized watcher
	rw, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFn})
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
			if stopFn(cond) {
				return nil
			}
		}
	}
}
