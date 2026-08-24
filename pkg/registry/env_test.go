package registry

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRegisterNamespaceEnv(t *testing.T) {
	RegisterNamespaceEnv("testns-env", EnvRequirement{
		Key:         "TESTNS_ENV_TOKEN",
		Description: "test token",
		Scope:       "all routes",
	})

	reqs := NamespaceEnvReqs("testns-env")
	if len(reqs) != 1 || reqs[0].Key != "TESTNS_ENV_TOKEN" {
		t.Fatalf("unexpected reqs: %+v", reqs)
	}

	// Unset -> not configured
	statuses, ok := AllEnvStatuses()["testns-env"]
	if !ok || len(statuses) != 1 {
		t.Fatalf("expected testns-env in statuses")
	}
	if statuses[0].Configured {
		t.Error("expected Configured=false when env unset")
	}

	// Set -> configured
	t.Setenv("TESTNS_ENV_TOKEN", "secret-value")
	statuses = AllEnvStatuses()["testns-env"]
	if !statuses[0].Configured {
		t.Error("expected Configured=true when env set")
	}

	// Values must never leak into status output
	b, err := json.Marshal(statuses)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "secret-value") {
		t.Errorf("env value leaked into status JSON: %s", b)
	}
}
