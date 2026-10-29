//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	cliBinaryOnce sync.Once
	cliBinaryPath string
	cliBinaryErr  error
)

// cliBinary builds the lok CLI once per test run and returns its path. The CLI
// runs in its own process, so these tests do not touch the shared office.
func cliBinary(t *testing.T) string {
	t.Helper()

	cliBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "lok-cli-")
		if err != nil {
			cliBinaryErr = err
			return
		}

		cliBinaryPath = filepath.Join(dir, "lok")

		out, err := exec.Command("go", "build", "-o", cliBinaryPath, "../../cmd/lok").CombinedOutput()
		if err != nil {
			cliBinaryErr = &buildError{err: err, out: string(out)}
		}
	})

	if cliBinaryErr != nil {
		t.Fatalf("building the lok CLI: %v", cliBinaryErr)
	}

	return cliBinaryPath
}

type buildError struct {
	err error
	out string
}

func (e *buildError) Error() string { return e.err.Error() + ": " + e.out }

// seededSetting is a registrymodifications.xcu entry a caller seeds into the
// profile before lok starts.
const seededSetting = `<item oor:path="/org.openoffice.Office.Common/Security/Scripting"><prop oor:name="BlockUntrustedRefererLinks" oor:op="fuse"><value>true</value></prop></item>`

// TestCLI_UserProfilePathKeepsSeededSettings pins the contract callers rely on
// to configure LibreOffice: a registrymodifications.xcu written into
// --user-profile-path before the run survives profile establishment, and the
// directory stays on disk after exit.
func TestCLI_UserProfilePathKeepsSeededSettings(t *testing.T) {
	input := testdataPath(t, "document.docx")

	profileDir := t.TempDir()
	userDir := filepath.Join(profileDir, "user")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}

	xcu := `<?xml version="1.0" encoding="UTF-8"?>
<oor:items xmlns:oor="http://openoffice.org/2001/registry" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  ` + seededSetting + `
</oor:items>
`
	xcuPath := filepath.Join(userDir, "registrymodifications.xcu")
	if err := os.WriteFile(xcuPath, []byte(xcu), 0o600); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(t.TempDir(), "out.pdf")

	out, err := exec.Command(cliBinary(t),
		"--input-path", input,
		"--output-path", output,
		"--user-profile-path", profileDir,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("lok failed: %v: %s", err, out)
	}

	assertValidPDF(t, output)

	data, err := os.ReadFile(xcuPath)
	if err != nil {
		t.Fatalf("the profile was not kept on exit: %v", err)
	}

	if !strings.Contains(string(data), `oor:name="BlockUntrustedRefererLinks"`) {
		t.Errorf("the seeded setting did not survive profile establishment:\n%s", data)
	}
}
