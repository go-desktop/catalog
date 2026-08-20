package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAndReadInventory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")
	in := Inventory{
		"zeta":  {{Org: "zeta", Name: "b"}, {Org: "zeta", Name: "a"}},
		"alpha": {{Org: "alpha", Name: "x"}},
	}
	if err := WriteInventory(path, in); err != nil {
		t.Fatalf("WriteInventory: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted output is what keeps a regeneration's diff limited to what GitHub
	// actually changed, instead of reshuffling on every run.
	if strings.Index(string(raw), `"alpha"`) > strings.Index(string(raw), `"zeta"`) {
		t.Error("organisations are not sorted")
	}
	if strings.Index(string(raw), `"a"`) > strings.Index(string(raw), `"b"`) {
		t.Error("repositories are not sorted within an organisation")
	}

	out, err := ReadInventory(path)
	if err != nil {
		t.Fatalf("ReadInventory: %v", err)
	}
	if len(out) != 2 || out["zeta"][0].Name != "a" {
		t.Errorf("round trip = %+v", out)
	}
}

func TestWriteInventoryFailures(t *testing.T) {
	if err := WriteInventory(filepath.Join(t.TempDir(), "no", "such", "dir.json"), Inventory{"a": nil}); err == nil {
		t.Error("WriteInventory should fail when the directory does not exist")
	}
	// An inventory cannot fail to encode, so the encode branch is exercised
	// through the helper with a value JSON cannot represent.
	if err := writeJSON(filepath.Join(t.TempDir(), "x.json"), make(chan int)); err == nil {
		t.Error("writeJSON should fail on an unencodable value")
	} else if !strings.Contains(err.Error(), "encode") {
		t.Errorf("error = %q, want it to mention encoding", err)
	}
}

func TestReadInventoryFailures(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadInventory(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("ReadInventory should fail on a missing file")
	}
	junk := filepath.Join(dir, "junk.json")
	if err := os.WriteFile(junk, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadInventory(junk); err == nil {
		t.Error("ReadInventory should fail on malformed JSON")
	}
	// A truncated inventory would otherwise regenerate the map into a blank
	// page without anything failing.
	empty := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(empty, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadInventory(empty); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("ReadInventory(empty) = %v, want an 'is empty' error", err)
	}
}
