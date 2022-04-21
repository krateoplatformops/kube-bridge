package secrets

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/krateoplatformops/kube-bridge/pkg/handlers"
	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func DeleteOne(cfg *rest.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := zerolog.Ctx(r.Context())

		kc, err := kubernetes.Secrets(cfg)
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		params := mux.Vars(r)

		lst, err := kc.List(params["namespace"], metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", handlers.CreatedByLabel, handlers.CreatedByValue),
		})
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusExpectationFailed)
			return
		}

		var s *corev1.Secret
		for _, el := range lst.Items {
			if strings.EqualFold(params["name"], el.Name) {
				s = el.DeepCopy()
				break
			}
		}

		if s == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		err = kc.Delete(params["name"], params["namespace"], metav1.DeleteOptions{})
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
