package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const standardIndex = `<?xml version="1.0" encoding="UTF-8"?>
<library:library xmlns:library="http://openoffice.org/2000/library" library:name="Standard" library:readonly="false" library:passwordprotected="false">
 <library:element library:name="Module1"/>
</library:library>
`

func TestRegisterModule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script.xlb")
	if err := os.WriteFile(path, []byte(standardIndex), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := registerModule(path); err != nil {
		t.Fatalf("registerModule: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(got), `library:name="Module1"`) {
		t.Error("existing Module1 element was removed")
	}

	if !strings.Contains(string(got), `library:name="LokGeometry"`) {
		t.Error("LokGeometry element was not added")
	}
}

func TestRegisterModule_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script.xlb")
	if err := os.WriteFile(path, []byte(standardIndex), 0o600); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := registerModule(path); err != nil {
			t.Fatalf("registerModule: %v", err)
		}
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if n := strings.Count(string(got), `library:name="LokGeometry"`); n != 1 {
		t.Errorf("expected exactly one LokGeometry element, got %d", n)
	}
}
