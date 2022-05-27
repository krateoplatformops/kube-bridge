package modules

import (
	"context"
	"time"
)

const (
	trIdKey = "transactionId"
)

func trId(ctx context.Context) string {
	trId, ok := ctx.Value(trIdKey).(string)
	if ok {
		return trId
	}

	return ""
}

type valueOnlyContext struct{ context.Context }

func (valueOnlyContext) Deadline() (deadline time.Time, ok bool) { return }
func (valueOnlyContext) Done() <-chan struct{}                   { return nil }
func (valueOnlyContext) Err() error                              { return nil }
