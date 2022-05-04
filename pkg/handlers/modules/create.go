package modules

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
	"github.com/krateoplatformops/kube-bridge/pkg/handlers/utils"
	"github.com/krateoplatformops/kube-bridge/pkg/support"
	"github.com/rs/zerolog"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func Create(cfg *rest.Config, bus eventbus.Bus) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := zerolog.Ctx(r.Context())

		var sd payload
		err := utils.DecodeJSONBody(w, r, &sd)
		if err != nil {
			log.Warn().Msg(err.Error())

			var mr *utils.MalformedRequest
			if errors.As(err, &mr) {
				http.Error(w, mr.Msg, mr.Status)
			} else {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		pkgObj, _, err := decodeModulePackage(sd.Package)
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Info().Str("name", pkgObj.GetName()).Msg("decoded package data")

		clmObj, clmGVK, err := decodeModuleClaim(sd.Claim)
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Info().
			Str("group", clmGVK.Group).
			Str("version", clmGVK.Version).
			Str("kind", clmGVK.Kind).
			Str("name", clmObj.GetName()).
			Msg("decoded claim data")

		pci := &packageAndClaimInfo{
			pkgObj: pkgObj,
			clmGVK: clmGVK,
			clmObj: clmObj,
		}

		go func() {
			err = installPackageAndClaim(r.Context(), bus, cfg, pci)
			if err != nil {
				log.Error().Msg(err.Error())
				bus.Publish(support.ErrorNotification(r.Context(), support.ReasonFailure, err))
				//http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			msg := fmt.Sprintf("package: %s and claim: %s successfully installed", pkgObj.GetName(), clmObj.GetName())
			bus.Publish(support.InfoNotification(r.Context(), support.ReasonSuccess, msg))
		}()

		w.WriteHeader(http.StatusOK)
	})
}

type packageAndClaimInfo struct {
	pkgObj *unstructured.Unstructured
	clmGVK *schema.GroupVersionKind
	clmObj *unstructured.Unstructured
}

func installPackageAndClaim(ctx context.Context, bus eventbus.Bus, cfg *rest.Config, pci *packageAndClaimInfo) error {
	log := zerolog.Ctx(ctx)
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	err = createOrUpdateResourceFromUnstructured(ctx, bus, cfg, dc, pci.pkgObj)
	if err != nil {
		return err
	}

	crdi := buildCRDInfo(pci.clmGVK)

	msg := fmt.Sprintf("Waiting for Resource (apiVersion: %s, kind: %s)", crdi.APIVersion, crdi.Spec.Names.Kind)
	bus.Publish(support.InfoNotification(ctx, support.ReasonWaitForResource, msg))

	log.Info().
		Str("apiVersion", crdi.APIVersion).
		Str("kind", crdi.Spec.Names.Kind).
		Str("plurals", crdi.Spec.Names.Plural).
		Msg("Waiting for CRD")
	err = waitForCRDs(cfg, []*apiextensionsv1.CustomResourceDefinition{crdi})
	if err != nil {
		return err
	}
	log.Info().
		Str("apiVersion", crdi.APIVersion).
		Str("kind", crdi.Spec.Names.Kind).
		Str("plurals", crdi.Spec.Names.Plural).
		Msg("CRD ready")

	msg = fmt.Sprintf("Resource ready (apiVersion: %s, kind: %s)", crdi.APIVersion, crdi.Spec.Names.Kind)
	bus.Publish(support.InfoNotification(ctx, support.ReasonSuccess, msg))

	return createOrUpdateResourceFromUnstructured(ctx, bus, cfg, dc, pci.clmObj)
}

type payload struct {
	Claim    string `json:"claim"`
	Package  string `json:"package"`
	Encoding string `json:"encoding"`
}
