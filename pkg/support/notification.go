package support

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
	"github.com/rs/zerolog"
)

const (
	NotificationEventID = eventbus.EventID("notify.event")
)

func InfoNotification(msg string) *Notification {
	return &Notification{
		Level:   "info",
		Service: ServiceName,
		Time:    time.Now().Unix(),
		Message: msg,
	}
}

func ErrorNotification(err error) *Notification {
	return &Notification{
		Level:   "error",
		Service: ServiceName,
		Time:    time.Now().Unix(),
		Error:   err.Error(),
	}
}

type Notification struct {
	Level   string `json:"level"`
	Time    int64  `json:"time"`
	Service string `json:"service"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (e *Notification) EventID() eventbus.EventID {
	return NotificationEventID
}

func NotificationDispatcher(addr string, log *zerolog.Logger) eventbus.EventHandler {

	return func(e eventbus.Event) {
		if e.EventID() != NotificationEventID {
			return
		}

		go func() {
			evt := e.(*Notification)

			dat, err := json.Marshal(evt)
			if err != nil {
				log.Error().Err(err).Msg("")
			}

			// TODO POST event
			fmt.Fprintln(os.Stderr, string(dat))
		}()
	}
}
