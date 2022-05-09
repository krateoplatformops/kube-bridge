package handlers

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
	"github.com/krateoplatformops/kube-bridge/pkg/support"
)

func HealtHandler(healthy *int32, bus eventbus.Bus) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(healthy) == 1 {
			go func() {
				msg := fmt.Sprintf("Ping received! (%s)", r.RemoteAddr)
				bus.Publish(support.InfoNotification(r.Context(), support.ReasonSuccess, msg))
			}()

			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}
