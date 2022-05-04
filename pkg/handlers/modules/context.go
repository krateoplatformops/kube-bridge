package modules

import "context"

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
