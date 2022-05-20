package handlers

import (
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/krateoplatformops/kube-bridge/pkg/support"
)

func HealtHandler(healthy *int32, version string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(healthy) == 1 {
			data := map[string]string{
				"name":    support.ServiceName,
				"version": version,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(data)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}
