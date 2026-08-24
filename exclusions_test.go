package catalog

import (
	"errors"
	"strings"
	"testing"
	"testing/iotest"
)

func TestLoadExclusions(t *testing.T) {
	e, err := LoadExclusions(strings.NewReader(`
# Retired from the map, kept on GitHub.
go-example

  Mixed-Case-Org  
`))
	if err != nil {
		t.Fatalf("LoadExclusions: %v", err)
	}
	if len(e) != 2 {
		t.Fatalf("read %d entries, want 2: %v", len(e), e)
	}
	// A comment is not a login, and a blank line is not an organisation called "".
	if e.Excludes("#") || e.Excludes("") {
		t.Error("a comment or a blank line became an entry")
	}
	// GitHub logins are case-insensitive, so the file must not have to match the
	// casing the API happens to return.
	for _, org := range []string{"go-example", "GO-EXAMPLE", "mixed-case-org", "Mixed-Case-Org"} {
		if !e.Excludes(org) {
			t.Errorf("Excludes(%q) = false, want true", org)
		}
	}
	if e.Excludes("go-gfx") {
		t.Error("Excludes(go-gfx) = true, want false")
	}
}

// A nil set excludes nothing, which is what makes Skip optional on the client.
func TestExclusionsNil(t *testing.T) {
	var e Exclusions
	if e.Excludes("go-gfx") {
		t.Error("the nil set excluded something")
	}
}

func TestLoadExclusionsError(t *testing.T) {
	_, err := LoadExclusions(iotest.ErrReader(errors.New("boom")))
	if err == nil || !strings.Contains(err.Error(), "read exclusions") {
		t.Errorf("LoadExclusions(failing reader) = %v, want a read error", err)
	}
}
