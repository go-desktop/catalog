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
	// The docs site is not a counted repository -- but it IS a cell the map
	// prints, so it belongs under pages rather than under repositories.
	if want := []string{"go-gfx/docs"}; !equal(d.NewPages, want) {
		t.Errorf("NewPages = %v, want %v", d.NewPages, want)
	}
	if len(d.GonePages) != 0 {
		t.Errorf("GonePages = %v, want none", d.GonePages)
	}
	if d.Empty() {
		t.Error("Empty() on a drift with five findings")
	}
	got := d.Report()
	for _, want := range []string{
		"+ go-arrives", "- go-leaves", "+ go-gfx/raster", "- go-gfx/colour",
		"+ go-gfx/docs",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Report() is missing %q:\n%s", want, got)
		}
	}
	// A private repository changes no cell on any page.
	if strings.Contains(got, "secret") {
		t.Errorf("Report() names a repository nothing publishes:\n%s", got)
	}
	// And the docs site is not counted as a repository gained.
	if strings.Contains(got, "+ go-gfx/docs\n  + ") || strings.Contains(strings.SplitN(got, "pages published", 2)[0], "go-gfx/docs") {
		t.Errorf("the docs site was reported as a repository:\n%s", got)
	}
}

// A landing page or a documentation site appearing is the finding that has no
// count behind it: no repository the map counts has moved, every rule still
// reconciles, and the only thing wrong is a dash in a cell that should now be a
// link. Losing one is the same in reverse.
func TestCompareInventoriesFindsPublishedPages(t *testing.T) {
	was := Inventory{"go-a": {{Org: "go-a", Name: "lib"}, {Org: "go-a", Name: "go-a.github.io"}},
		"go-b": {{Org: "go-b", Name: "lib"}, {Org: "go-b", Name: "docs"}}}
	now := Inventory{"go-a": {{Org: "go-a", Name: "lib"}, {Org: "go-a", Name: "docs"}},
		"go-b": {{Org: "go-b", Name: "lib"}}}
	d := CompareInventories(was, now)
	if want := []string{"go-a/docs"}; !equal(d.NewPages, want) {
		t.Errorf("NewPages = %v, want %v", d.NewPages, want)
	}
	if want := []string{"go-a/site", "go-b/docs"}; !equal(d.GonePages, want) {
		t.Errorf("GonePages = %v, want %v", d.GonePages, want)
	}
	// None of this is a repository the map counts.
	if len(d.NewRepos) != 0 || len(d.GoneRepos) != 0 {
		t.Errorf("publishing repositories were counted: new=%v gone=%v", d.NewRepos, d.GoneRepos)
	}
	got := d.Report()
	for _, want := range []string{"published since", "published no longer", "- go-a/site", "+ go-a/docs"} {
		if !strings.Contains(got, want) {
			t.Errorf("Report() is missing %q:\n%s", want, got)
		}
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
