package ui

import (
	"io/fs"
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
	if _, err := fs.Stat(FS(), "inherent.css"); err != nil {
		t.Fatalf("inherent.css not in ui.FS(): %v", err)
	}
}
