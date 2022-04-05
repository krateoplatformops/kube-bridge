package platform

import (
	"context"
	"embed"
	"fmt"

	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
	"github.com/krateoplatformops/kube-bridge/pkg/helm"
	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	"github.com/krateoplatformops/kube-bridge/pkg/support"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	chartArchive     = "assets/crossplane-1.7.0.tgz"
	chartReleaseName = "crossplane"
)

//go:embed assets/*
var assetsFS embed.FS

func installCrossplaneEventually(dc dynamic.Interface, opts InitOptions) error {
	ok, err := isCrossplaneInstalled(dc)
	if err != nil {
		return err
	}

	if ok {
		opts.Bus.Publish(support.InfoNotification("crossplane already installed"))
		return nil
	}

	opts.Bus.Publish(support.InfoNotification("installing crossplane chart..."))
	err = installCrossplaneChart(opts.Config, opts.Bus, opts.Verbose)
	if err != nil {
		return err
	}
	opts.Bus.Publish(support.InfoNotification("crossplane chart successfully installed."))

	return waitForCrossplaneReady(dc)
}

func installCrossplaneChart(rc *rest.Config, bus eventbus.Bus, verbose bool) error {
	fp, err := assetsFS.Open(chartArchive)
	if err != nil {
		return err
	}
	defer fp.Close()

	opts := &helm.InstallOptions{
		Namespace:   kubernetes.CrossplaneSystemNamespace,
		ReleaseName: chartReleaseName,
		ChartSource: fp,
		ChartValues: map[string]interface{}{
			"securityContextCrossplane": map[string]interface{}{
				"runAsUser":  nil,
				"runAsGroup": nil,
			},
			"securityContextRBACManager": map[string]interface{}{
				"runAsUser":  nil,
				"runAsGroup": nil,
			},
		},
		LogFn: func(format string, v ...interface{}) {
			if verbose && bus != nil {
				msg := fmt.Sprintf(format, v...)
				bus.Publish(support.InfoNotification(msg))
			}
		},
	}

	return helm.Install(rc, opts)
}

func isCrossplaneInstalled(dc dynamic.Interface) (bool, error) {
	sel, err := labels.Parse("app=crossplane")
	if err != nil {
		return false, err
	}

	return podExists(dc, sel)
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
