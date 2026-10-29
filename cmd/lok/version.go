package main

import "runtime/debug"

// version is the release version. The release build sets it with
// -ldflags "-X main.version=...".
var version string

// versionString reports the release version, else the module version a
// `go install ...@vX.Y.Z` build records, else "devel" for a source checkout.
func versionString() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "devel"
}
