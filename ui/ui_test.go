package ui

import (
	"io/fs"
	"strings"
	"testing"
)

func TestPropsBuildsPairs(t *testing.T) {
	fm := Funcs()
	fn, ok := fm["props"].(func(...string) (Props, error))
	if !ok {
		t.Fatalf("props missing or wrong signature")
	}
	p, err := fn("variant", "primary", "type", "submit")
	if err != nil {
		t.Fatal(err)
	}
	if p["variant"] != "primary" || p["type"] != "submit" {
		t.Fatalf("props = %v", p)
	}
}

func TestPropsRejectsOddCount(t *testing.T) {
	if _, err := props("a"); err == nil {
		t.Fatal("expected error for odd argument count")
	}
}

func TestFSServesBaseCSS(t *testing.T) {
	b, err := fs.ReadFile(FS(), "base.css")
	if err != nil {
		t.Fatalf("base.css not in ui.FS(): %v", err)
	}
	if !strings.Contains(string(b), "Inherent.css") {
		t.Fatalf("ui base stylesheet is not the vendored inherent.css")
	}
}