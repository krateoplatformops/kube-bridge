package support

import (
	"os"
	"strconv"
	"strings"
)

const (
	ServiceName = "kube-bridge"
)

func EnvString(key, defaultValue string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return val
}

func EnvInt(key string, defaultValue int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	n := strings.TrimSpace(val)
	if len(n) == 0 {
		return defaultValue
	}

	res, err := strconv.Atoi(n)
	if err != nil {
		return defaultValue
	}
	return res
}

func EnvBool(key string, defaultValue bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	res, err := strconv.ParseBool(strings.TrimSpace(val))
	if err != nil {
		return defaultValue
	}
	return res
}
