package support

import (
	"os"
	"testing"
)

func TestEnv(t *testing.T) {
	exp := "krateo@kiratech.it"

	os.Setenv("GUMROADCB_SMTP_FROM", exp)

	act := Env("SMTP_FROM", "")
	if act != exp {
		t.Fatalf("expected: %s, got: %s", exp, act)
	}
}
