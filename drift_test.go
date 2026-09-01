package catalog

import (
	"strings"
	"testing"
)

func TestCompareInventoriesFindsNothingInAnUnchangedEcosystem(t *testing.T) {
	inv := Inventory{
		"go-gfx":  {{Org: "go-gfx", Name: "gfx"}, {Org: "go-gfx", Name: "docs"}},
		"go-held": {},
	}
	d := CompareInventories(inv, inv)
	if !d.Empty() {
		t.Errorf("an inventory drifted from itself: %+v", d)
	}
	if got := d.Report(); !strings.HasPrefix(got, "no drift") {
		t.Errorf("Report() = %q, want it to say there is none", got)
	}
}

func TestCompareInventories(t *testing.T) {
	was := Inventory{
		"go-gfx":    {{Org: "go-gfx", Name: "gfx"}, {Org: "go-gfx", Name: "colour"}},
		"go-leaves": {{Org: "go-leaves", Name: "leaf"}},
		"go-held":   {},
	}
	now := Inventory{
		"go-gfx": {
			{Org: "go-gfx", Name: "gfx"},
			// colour is gone, raster is new, and neither the docs site nor a
			// private repository is a change anyone reading the map can see.
			{Org: "go-gfx", Name: "raster"},
			{Org: "go-gfx", Name: "docs"},
			{Org: "go-gfx", Name: "secret", Private: true},
		},
		"go-held":    {},
		"go-arrives": {{Org: "go-arrives", Name: "thing"}},
	}
	d := CompareInventories(was, now)
	if want := []string{"go-arrives"}; !equal(d.NewOrgs, want) {
		t.Errorf("NewOrgs = %v, want %v", d.NewOrgs, want)
	}
	if want := []string{"go-leaves"}; !equal(d.GoneOrgs, want) {
		t.Errorf("GoneOrgs = %v, want %v", d.GoneOrgs, want)
	}
	// The vanished organisation's repository is NOT listed again underneath it.
	if want := []string{"go-gfx/raster"}; !equal(d.NewRepos, want) {
		t.Errorf("NewRepos = %v, want %v", d.NewRepos, want)
	}
	if want := []string{"go-gfx/colour"}; !equal(d.GoneRepos, want) {
		t.Errorf("GoneRepos = %v, want %v", d.GoneRepos, want)
	}
	if d.Empty() {
		t.Error("Empty() on a drift with four findings")
	}
	got := d.Report()
	for _, want := range []string{
		"+ go-arrives", "- go-leaves", "+ go-gfx/raster", "- go-gfx/colour",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Report() is missing %q:\n%s", want, got)
		}
	}
	// A repository that only moved in or out of the publishing set must not be
	// reported: nothing a reader sees changed.
	if strings.Contains(got, "docs") || strings.Contains(got, "secret") {
		t.Errorf("Report() names a repository the map does not count:\n%s", got)
	}
}

// A drift with findings of only one kind must print only that section, so that
// an empty heading never invites someone to look for what is not there.
func TestDriftReportOmitsEmptySections(t *testing.T) {
	d := Drift{NewRepos: []string{"go-gfx/raster"}}
	got := d.Report()
	if strings.Contains(got, "organisations") || strings.Contains(got, "are gone") {
		t.Errorf("Report() printed a section with nothing in it:\n%s", got)
	}
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
