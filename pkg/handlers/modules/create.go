package modules

import (
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/krateoplatformops/kube-bridge/pkg/handlers/utils"
	"github.com/rs/zerolog"
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

		pkg, err := base64.StdEncoding.DecodeString(sd.Package)
		if err != nil {
			log.Warn().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		clm, err := base64.StdEncoding.DecodeString(sd.Claim)
		if err != nil {
			log.Warn().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = installModulePackage(cfg, pkg)
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = installModuleClaims(cfg, clm)
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
