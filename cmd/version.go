package cmd

import (
	"fmt"
	"runtime/debug"
)

// Version resolves the version string to report. `go install pkg@version`
// stamps bi.Main.Version automatically; a local build falls back to the VCS
// revision Go records in bi.Settings.
func Version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}

	var revision string
	var modified bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return formatRevision(revision, modified)
}

// formatRevision renders a VCS revision as a short version string.
func formatRevision(rev string, modified bool) string {
	if rev == "" {
		return "devel"
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	if modified {
		rev += "+dirty"
	}
	return rev
}

// RunVersion prints the resolved version to stdout.
func RunVersion(_ []string) error {
	fmt.Println("park", Version())
	return nil
}
