package platform

import (
	"fmt"

	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	"github.com/krateoplatformops/kube-bridge/pkg/support"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type InitOptions struct {
	Config  *rest.Config
	Bus     eventbus.Bus
	Verbose bool
}

func Init(opts InitOptions) error {
	dc, err := dynamic.NewForConfig(opts.Config)
	if err != nil {
		return err
	}

	// Create Crossplane namespace
	err = createNamespace(dc, kubernetes.CrossplaneSystemNamespace)
	if err != nil {
		return err
	}
	opts.Bus.Publish(support.InfoNotification(fmt.Sprintf("namespace '%s' successfully created", kubernetes.CrossplaneSystemNamespace)))

	// Install Crossplane
	err = installCrossplaneEventually(dc, opts)
	if err != nil {
		return err
	}

	// Install controller config
	err = createControllerConfig(dc)
	if err != nil {
		return err
	}

	// Install Crossplane provider helm
	err = installCrossplaneProviderHelmEventually(dc)
	if err != nil {
		return err
	}

	// Install crossplane provider kubernetes
	err = installCrossplaneProviderKubernetesEventually(dc)
	if err != nil {
		return err
	}
	opts.Bus.Publish(support.InfoNotification("Runtime successfully installed"))

	opts.Bus.Publish(support.InfoNotification("Creating roles bindings..."))
	err = createRoleBindingForCrossplaneProviders(dc)
	if err != nil {
		return err
	}
	opts.Bus.Publish(support.InfoNotification("Roles bindings successfully created"))

	opts.Bus.Publish(support.InfoNotification(fmt.Sprintf("Creating namespace %s)", kubernetes.KrateoSystemNamespace)))
	createNamespace(dc, kubernetes.KrateoSystemNamespace)
	opts.Bus.Publish(support.InfoNotification(fmt.Sprintf("Namespace '%s' created", kubernetes.KrateoSystemNamespace)))

	return nil
}

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
