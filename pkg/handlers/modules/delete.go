package modules

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
	"github.com/krateoplatformops/kube-bridge/pkg/handlers/utils"
	"github.com/krateoplatformops/kube-bridge/pkg/support"
	"github.com/rs/zerolog"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func Delete(cfg *rest.Config, bus eventbus.Bus) http.Handler {
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
			ctx := valueOnlyContext{r.Context()}

			err = deletePackageAndClaim(ctx, bus, cfg, pci)
			if err != nil {
				log.Error().Msg(err.Error())
				bus.Publish(support.ErrorNotification(ctx, support.ReasonFailure, err))
				return
			}

			msg := fmt.Sprintf("package: %s and claim: %s successfully deleted", pkgObj.GetName(), clmObj.GetName())
			bus.Publish(support.InfoNotification(ctx, support.ReasonSuccess, msg))
		}()

		w.WriteHeader(http.StatusOK)
	})
}

func deletePackageAndClaim(ctx context.Context, bus eventbus.Bus, cfg *rest.Config, pci *packageAndClaimInfo) error {
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	err = deleteResourceFromUnstructured(ctx, bus, cfg, dc, pci.clmObj)
	if err != nil {
		return err
	}

	return createOrUpdateResourceFromUnstructured(ctx, bus, cfg, dc, pci.pkgObj)
}
