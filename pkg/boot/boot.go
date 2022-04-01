package boot

import (
	"fmt"

	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	"github.com/krateoplatformops/kube-bridge/pkg/support"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type BootOptions struct {
	Config  *rest.Config
	Bus     eventbus.Bus
	Verbose bool
}

func Run(opts BootOptions) error {
	dc, err := dynamic.NewForConfig(opts.Config)
	if err != nil {
		return err
	}

	// Create Crossplane namespace
	err = createNamespace(dc, kubernetes.CrossplaneSystemNamespace)
	if err != nil {
		return err
	}

	// Install Crossplane
	opts.Bus.Publish(support.InfoNotification("Installing Runtime..."))
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

func installCrossplaneEventually(dc dynamic.Interface, opts BootOptions) error {
	ok, err := isCrossplaneInstalled(dc)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	err = installCrossplaneChart(opts.Config, opts.Bus, opts.Verbose)
	if err != nil {
		return err
	}

	return waitForCrossplaneReady(dc)
}
