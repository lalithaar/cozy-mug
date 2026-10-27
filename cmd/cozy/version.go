package main

import (
	"runtime/debug"
	"strings"
)

// modulePath is the framework's import path; scaffolds require it.
const modulePath = "github.com/lalithaar/cozy-mug"

// version is injected at build time for release binaries:
//
//	go build -ldflags "-X main.version=v0.0.13" ./cmd/cozy
//
// It stays empty for plain `go build` / `go run` / `go test` builds, which fall
// back to the module version recorded in the build info (populated by
// `go install github.com/lalithaar/cozy-mug/cmd/cozy@vX.Y.Z`). A development
// build therefore reports "" and must not pin a version it cannot know.
var version = ""

// resolvedVersion returns this build's framework version (vX.Y.Z, possibly a
// pseudo-version), or "" when it cannot be determined.
func resolvedVersion() string {
	if v := usableVersion(version); v != "" {
		return v
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	// `go install <module>/cmd/cozy@vX.Y.Z` makes the framework the main module.
	if info.Main.Path == modulePath {
		if v := usableVersion(info.Main.Version); v != "" {
			return v
		}
	}
	// Building inside a checkout reports "(devel)" for the main module, so the
	// framework shows up as a dependency only when a consumer builds the CLI.
	for _, dep := range info.Deps {
		if dep.Path == modulePath {
			return usableVersion(dep.Version)
		}
	}
	return ""
}

// usableVersion accepts a released module version (vX.Y.Z or vX.Y.Z-rc1) and
// rejects what a local build reports: "" , "(devel)", pseudo-versions derived
// from a checkout, and anything carrying build metadata such as "+dirty". A
// pseudo-version is not resolvable from the module proxy, so it must never be
// written into a scaffold's go.mod.
func usableVersion(v string) string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "v") || strings.ContainsAny(v, "+ ") {
		return ""
	}
	body := strings.TrimPrefix(v, "v")
	core := body
	if i := strings.IndexByte(body, '-'); i >= 0 {
		pre := body[i+1:]
		core = body[:i]
		// Pseudo-versions look like -0.yyyymmddhhmmss-<rev> or
		// -yyyymmddhhmmss-<rev>; release prereleases look like -rc1.
		if strings.HasPrefix(pre, "0.") || strings.HasPrefix(pre, "0-") || isPseudoPre(pre) {
			return ""
		}
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return ""
	}
	for _, p := range parts {
		if p == "" || strings.Trim(p, "0123456789") != "" {
			return ""
		}
	}
	return v
}

// isPseudoPre reports whether a prerelease segment starts with a
// yyyymmddhhmmss timestamp, the other pseudo-version shape.
func isPseudoPre(pre string) bool {
	head := pre
	if i := strings.LastIndexByte(pre, '-'); i >= 0 {
		head = pre[:i]
	} else {
		return false
	}
	if len(head) != 14 || strings.Trim(head, "0123456789") != "" {
		return false
	}
	return true
}
