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
	err = installCrossplaneProviderEventually(dc, providerHelm())
	if err != nil {
		return err
	}
	opts.Bus.Publish(support.InfoNotification(fmt.Sprintf("crossplane provider '%s' successfully installed", providerHelm())))

	// Install crossplane provider kubernetes
	err = installCrossplaneProviderEventually(dc, providerKubernetes())
	if err != nil {
		return err
	}
	opts.Bus.Publish(support.InfoNotification(fmt.Sprintf("crossplane provider '%s' successfully installed", providerKubernetes())))

	err = createRoleBindingForCrossplaneProviders(dc)
	if err != nil {
		return err
	}
	opts.Bus.Publish(support.InfoNotification("roles bindings successfully created"))

	createNamespace(dc, kubernetes.KrateoSystemNamespace)
	opts.Bus.Publish(support.InfoNotification(fmt.Sprintf("namespace '%s' created", kubernetes.KrateoSystemNamespace)))

	return nil
}
