// Package version holds release metadata injected at link time (-ldflags -X).
package version

import (
	"runtime/debug"
	"strings"
)

// Version is the service / image tag (semver, sha, etc.). Set at link time, e.g.:
//
//	-X 'audit-log/internal/version.Version=v1.2.3'
//
// When empty, String() falls back to module or VCS info from the build.
var Version string

// String returns Version if set, otherwise build metadata, otherwise "0.0.0-dev".
func String() string {
	if v := strings.TrimSpace(Version); v != "" {
		return v
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if mv := strings.TrimSpace(bi.Main.Version); mv != "" && mv != "(devel)" {
			return mv
		}
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				rev := s.Value
				if len(rev) > 12 {
					rev = rev[:12]
				}
				return "0.0.0-dev+" + rev
			}
		}
	}
	return "0.0.0-dev"
}
