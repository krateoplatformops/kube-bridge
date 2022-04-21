package secrets

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/krateoplatformops/kube-bridge/pkg/handlers"
	"github.com/krateoplatformops/kube-bridge/pkg/handlers/utils"
	"github.com/krateoplatformops/kube-bridge/pkg/kubernetes"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
)

func Create(cfg *rest.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := zerolog.Ctx(r.Context())

		var sd secretData
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

		params := mux.Vars(r)

		s := newSecretObj(params["name"], params["namespace"], corev1.SecretTypeOpaque)
		err = addToSecret(s, &sd)
		if err != nil {
			log.Warn().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		kc, err := kubernetes.Secrets(cfg)
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = kc.Create(s.Namespace, s, metav1.CreateOptions{})
		if err != nil {
			log.Error().Msg(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "secret '%s' created", params["name"])
	})
}

type keyval struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

type keyvals []keyval

type secretData struct {
	Data keyvals `json:"data"`
}

// newSecretObj will create a new Secret Object given name, namespace and secretType
func newSecretObj(name, namespace string, secretType corev1.SecretType) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				handlers.CreatedByLabel: handlers.CreatedByValue,
			},
		},
		Type: secretType,
		Data: map[string][]byte{},
	}
}

// addToSecret adds the given key and data to the given secret,
// returning an error if the key is not valid or if the key already exists.
func addToSecret(secret *corev1.Secret, sd *secretData) error {
	for _, item := range sd.Data {
		if errs := validation.IsConfigMapKey(item.Key); len(errs) != 0 {
			return fmt.Errorf("%q is not valid key name for a Secret %s", item.Key, strings.Join(errs, ";"))
		}
		if _, entryExists := secret.Data[item.Key]; entryExists {
			return fmt.Errorf("cannot add key %s, another key by that name already exists", item.Key)
		}
		secret.Data[item.Key] = []byte(item.Val)
	}

	return nil
}
