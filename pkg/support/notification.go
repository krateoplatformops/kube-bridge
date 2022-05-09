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
	trIdKey             = "developmentId"
)

const (
	ReasonWaitForResource = "WaitForResource"
	ReasonSuccess         = "Success"
	ReasonFailure         = "Failure"
	ReasonResourceUpdated = "ResourceUpdated"
	ReasonResourceCreated = "ResourceCreated"
)

func InfoNotification(ctx context.Context, rsn, msg string) *Notification {
	ret := &Notification{
		Level:   "info",
		Source:  ServiceName,
		Time:    time.Now().Unix(),
		Reason:  rsn,
		Message: msg,
	}

	trId, ok := ctx.Value(trIdKey).(string)
	if ok {
		ret.TransactionId = trId
	}

	return ret
}

func ErrorNotification(ctx context.Context, rsn string, err error) *Notification {
	ret := &Notification{
		Level:   "error",
		Source:  ServiceName,
		Time:    time.Now().Unix(),
		Reason:  rsn,
		Message: err.Error(),
	}

	trId, ok := ctx.Value(trIdKey).(string)
	if ok {
		ret.TransactionId = trId
	}

	return ret
}

type Notification struct {
	Level         string `json:"level"`
	Time          int64  `json:"time"`
	Message       string `json:"message"`
	Source        string `json:"source"`
	Reason        string `json:"reason"`
	TransactionId string `json:"deploymentId"`
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
				fmt.Fprintf(os.Stderr, "reqId: %s - error: %s", evt.TransactionId, err.Error())
				return
			}
			fmt.Fprintf(os.Stderr, "==> %s <==", dat)

			ctx, cncl := context.WithTimeout(context.Background(), time.Second*40)
			defer cncl()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, bytes.NewBuffer(dat))
			if err != nil {
				fmt.Fprintf(os.Stderr, "reqId: %s - error: %s", evt.TransactionId, err.Error())
				return
			}
			req.Header.Set("Content-Type", "application/json")

			_, err = http.DefaultClient.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "reqId: %s - error: %s", evt.TransactionId, err.Error())
				return
			}
		}()
	}
}
