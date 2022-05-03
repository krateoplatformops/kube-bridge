package support

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/krateoplatformops/kube-bridge/pkg/eventbus"
)

const (
	NotificationEventID = eventbus.EventID("notify.event")
	reqIdKey            = "correlation_id"
)

func InfoNotification(ctx context.Context, msg string) *Notification {
	ret := &Notification{
		Level:   "info",
		Source:  ServiceName,
		Time:    time.Now().Unix(),
		Message: msg,
	}

	reqId, ok := ctx.Value(reqIdKey).(string)
	if ok {
		ret.CorrelationId = reqId
	}

	return ret
}

func ErrorNotification(ctx context.Context, err error) *Notification {
	ret := &Notification{
		Level:   "error",
		Source:  ServiceName,
		Time:    time.Now().Unix(),
		Message: err.Error(),
	}

	reqId, ok := ctx.Value(reqIdKey).(string)
	if ok {
		ret.CorrelationId = reqId
	}

	return ret
}

type Notification struct {
	Level         string `json:"level"`
	Time          int64  `json:"time"`
	Message       string `json:"message"`
	Source        string `json:"source"`
	Reason        string `json:"reason"`
	CorrelationId string `json:"transactionId"`
}

func (e *Notification) EventID() eventbus.EventID {
	return NotificationEventID
}

func NotificationDispatcher(addr string) eventbus.EventHandler {
	return func(e eventbus.Event) {
		if e.EventID() != NotificationEventID {
			return
		}

		go func() {
			evt := e.(*Notification)

			dat, err := json.Marshal(evt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "reqId: %s - error: %s", evt.CorrelationId, err.Error())
				return
			}
			fmt.Fprintf(os.Stderr, "==> %s <==", dat)

			ctx, cncl := context.WithTimeout(context.Background(), time.Second*40)
			defer cncl()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, bytes.NewBuffer(dat))
			if err != nil {
				fmt.Fprintf(os.Stderr, "reqId: %s - error: %s", evt.CorrelationId, err.Error())
				return
			}
			req.Header.Set("Content-Type", "application/json")

			_, err = http.DefaultClient.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "reqId: %s - error: %s", evt.CorrelationId, err.Error())
				return
			}
		}()
	}
}
