package boot

import (
	"embed"
	"fmt"

	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
	"github.com/krateoplatformops/kube-bridge/pkg/helm"
	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	"github.com/krateoplatformops/kube-bridge/pkg/support"
	"k8s.io/client-go/rest"
)

const (
	chartArchive     = "assets/crossplane-1.6.3.tgz"
	chartReleaseName = "crossplane"
)

//go:embed assets/*
var assetsFS embed.FS

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
