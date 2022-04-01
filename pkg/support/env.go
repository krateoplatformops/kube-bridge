package support

import (
	"fmt"
	"os"
	"strings"
)

const (
	ServiceName = "kube-bridge"
)

func Env(key, defaultValue string) string {
	full := fmt.Sprintf("%s_%s", strings.ToUpper(ServiceName), key)
	val, ok := os.LookupEnv(full)
	if !ok {
		return defaultValue
	}
	return val
}
