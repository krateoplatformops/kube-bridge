package modules

import (
	"errors"
	"net/http"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/krateoplatformops/kube-bridge/pkg/handlers/utils"
	"github.com/rs/zerolog"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func Create(cfg *rest.Config) http.Handler {
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
		log.Debug().Str("name", pkgObj.GetName()).Msg("decoded package data")

		clmObj, clmGVK, err := decodeModuleClaim(sd.Claim)
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Debug().
			Str("group", clmGVK.Group).
			Str("version", clmGVK.Version).
			Str("kind", clmGVK.Kind).
			Str("name", clmObj.GetName()).
			Msg("decoded claim data")

		dc, err := dynamic.NewForConfig(cfg)
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = createOrUpdateResourceFromUnstructured(cfg, dc, pkgObj)
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = waitForCRDs(cfg, []*apiextensionsv1.CustomResourceDefinition{buildCRDInfo(clmGVK)})
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = createOrUpdateResourceFromUnstructured(cfg, dc, clmObj)
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

type payload struct {
	Claim    string `json:"claim"`
	Package  string `json:"package"`
	Encoding string `json:"encoding"`
}
