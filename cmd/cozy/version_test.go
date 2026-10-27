package main

import (
	"strings"
	"testing"
)

func TestUsableVersion(t *testing.T) {
	cases := map[string]string{
		"v0.0.13":    "v0.0.13",
		"v0.1.0-rc1": "v0.1.0-rc1",
		"":           "",
		"(devel)":    "",
		"0.0.13":     "",
		"  v0.0.9 ":  "v0.0.9",
		// A checkout build stamps a pseudo-version; it is not proxy-resolvable.
		"v0.0.13-0.20261016124624-63d9358cf061":       "",
		"v0.0.13-0.20261016124624-63d9358cf061+dirty": "",
		"v0.0.13-20261016124624-63d9358cf061":         "",
		"v0.0.13+dirty":                               "",
	}
	for in, want := range cases {
		if got := usableVersion(in); got != want {
			t.Errorf("usableVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGomodFilePinsResolvedVersion(t *testing.T) {
	got := gomodFile("example.com/site", "v0.0.13")
	want := "module example.com/site\n\ngo 1.27\n\nrequire github.com/lalithaar/cozy-mug v0.0.13\n"
	if got != want {
		t.Fatalf("gomodFile with version:\n got %q\nwant %q", got, want)
	}
}

func TestGomodFileOmitsUnknownVersion(t *testing.T) {
	got := gomodFile("example.com/site", "")
	if strings.Contains(got, "require") {
		t.Fatalf("gomodFile without version must not invent a require line:\n%s", got)
	}
	if !strings.Contains(got, "module example.com/site") {
		t.Fatalf("gomodFile missing module line:\n%s", got)
	}
}

// A test binary is a development build, so there is no version to pin.
func TestResolvedVersionInTestBuild(t *testing.T) {
	if v := resolvedVersion(); v != "" {
		t.Fatalf("resolvedVersion() = %q in a test build, want \"\"", v)
	}
}
