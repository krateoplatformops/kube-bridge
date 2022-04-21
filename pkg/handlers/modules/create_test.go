package modules

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestB64(t *testing.T) {
	pkg, err := ioutil.ReadFile("../../../testdata/package.yaml")
	if err != nil {
		t.Fatal(err)
	}

	pkgEnc := base64.StdEncoding.EncodeToString(pkg)

	clm, err := ioutil.ReadFile("../../../testdata/claims.yaml")
	if err != nil {
		t.Fatal(err)
	}

	clmEnc := base64.StdEncoding.EncodeToString(clm)

	m := map[string]string{
		"package":  pkgEnc,
		"claims":   clmEnc,
		"encoding": "base64",
	}

	js, err := json.MarshalIndent(m, " ", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", js)
}
