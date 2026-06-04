// Package profile installs the lok Basic macro into a LibreOffice user profile.
//
// LibreOfficeKit cannot change a document's page geometry through a UNO
// dispatch, so geometry is applied by a Basic macro run via runMacro. The macro
// lives in the profile's Standard Basic library.
//
// LibreOffice regenerates the Standard library the first time a fresh profile
// initializes Basic, which would erase a pre-written macro. Install therefore
// first establishes the profile with a throwaway soffice run, then adds the
// macro to the established library, leaving the default Module1 intact.
package profile

import (
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed assets/LokGeometry.xba
var moduleXBA string

// MacroURL is the runMacro URL of the geometry macro.
const MacroURL = "macro:///Standard.LokGeometry.ApplyGeometry"

// Install establishes the profile at profileDir using the LibreOffice in
// programPath, then adds the geometry macro to its Standard Basic library.
func Install(programPath, profileDir string) error {
	if err := establish(programPath, profileDir); err != nil {
		return err
	}

	standardDir := filepath.Join(profileDir, "user", "basic", "Standard")
	if _, err := os.Stat(standardDir); err != nil {
		return fmt.Errorf("standard basic library not created by establish: %w", err)
	}

	if err := os.WriteFile(filepath.Join(standardDir, "LokGeometry.xba"), []byte(moduleXBA), 0o600); err != nil {
		return fmt.Errorf("writing macro module: %w", err)
	}

	if err := registerModule(filepath.Join(standardDir, "script.xlb")); err != nil {
		return fmt.Errorf("registering macro module: %w", err)
	}

	return nil
}

// establish runs soffice once with --terminate_after_init so LibreOffice writes
// the profile registry and the default Standard Basic library. After this the
// profile is established and a later init keeps any macro on disk.
func establish(programPath, profileDir string) error {
	abs, err := filepath.Abs(profileDir)
	if err != nil {
		return fmt.Errorf("resolving profile path: %w", err)
	}

	userInstallation := (&url.URL{Scheme: "file", Path: abs}).String()

	cmd := exec.Command( //nolint:gosec // soffice path is derived from the caller's program directory
		filepath.Join(programPath, "soffice"),
		"--headless",
		"--invisible",
		"--nologo",
		"--norestore",
		"--terminate_after_init",
		"-env:UserInstallation="+userInstallation,
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("establishing profile via soffice: %w: %s", err, out)
	}

	return nil
}

// registerModule adds the LokGeometry element to the Standard library index at
// scriptXLB, preserving the existing modules. It is a no-op if already present.
func registerModule(scriptXLB string) error {
	data, err := os.ReadFile(scriptXLB)
	if err != nil {
		return err
	}

	content := string(data)
	if strings.Contains(content, `library:name="LokGeometry"`) {
		return nil
	}

	const closing = "</library:library>"

	idx := strings.LastIndex(content, closing)
	if idx < 0 {
		return fmt.Errorf("unexpected library index format in %s", scriptXLB)
	}

	merged := content[:idx] + ` <library:element library:name="LokGeometry"/>` + "\n" + content[idx:]

	return os.WriteFile(scriptXLB, []byte(merged), 0o600)
}
