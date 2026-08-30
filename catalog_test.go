package catalog

import (
	"strings"
	"testing"
)

func buildFixture(t *testing.T, classification string, inv Inventory) (*Catalog, error) {
	t.Helper()
	cl, err := LoadClassification(strings.NewReader(classification))
	if err != nil {
		t.Fatalf("classification: %v", err)
	}
	return Build(cl, inv)
}

func fixtureInventory() Inventory {
	return Inventory{
		"go-widgets": {
			{Name: "toolkit"}, {Name: "painter"},
			{Name: "go-widgets.github.io"}, {Name: "docs"},
		},
		"go-gfx":       {{Name: "gfx"}},
		"go-ruby-json": {{Name: "json"}},
		// renovate-runner is org plumbing, so this organisation holds one library,
		// not two — the fixture keeps it precisely to pin that down.
		"go-ruby-stdlib":         {{Name: "stdlib"}, {Name: "renovate-runner"}},
		"go-quake2":              {},                                // reserved, empty
		"example-c":              {{Name: "c-fw"}},                  // not Go
		"go-desktop":             {{Name: "brand"}, {Name: "docs"}}, // infra only
		"all-private":            {{Name: "thing", Private: true}},  // invisible
		"go-ruby-empty-reserved": {},                                // a gem name with no code
	}
}

func TestBuild(t *testing.T) {
	c, err := buildFixture(t, goodClassification, fixtureInventory())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Two classified organisations plus two gem organisations.
	if c.TotalOrgs != 4 {
		t.Errorf("TotalOrgs = %d, want 4", c.TotalOrgs)
	}
	// go-widgets 2 + go-gfx 1 + json 1 + stdlib 1 (its renovate-runner is plumbing).
	if c.TotalRepos != 5 {
		t.Errorf("TotalRepos = %d, want 5", c.TotalRepos)
	}
	if got := c.GemRepos(); got != 2 {
		t.Errorf("GemRepos() = %d, want 2", got)
	}
	if len(c.Gems) != 2 || c.Gems[0].Name != "json" || c.Gems[1].Name != "stdlib" {
		t.Errorf("gems = %+v, want json then stdlib (sorted)", c.Gems)
	}
	if c.Gems[0].Org != "go-ruby-json" {
		t.Errorf("gem org = %q, want go-ruby-json", c.Gems[0].Org)
	}
	if c.GemsIntro == "" {
		t.Error("the gems introduction did not survive Build")
	}
	if len(c.Reserved) != 1 || len(c.NotGo) != 1 {
		t.Errorf("reserved/not-Go = %d/%d, want 1/1", len(c.Reserved), len(c.NotGo))
	}
	w, ok := c.Family("desktop")
	if !ok {
		t.Fatal("Family(desktop) not found")
	}
	if len(w.Orgs) != 1 {
		t.Fatalf("desktop has %d orgs, want 1", len(w.Orgs))
	}
	got := w.Orgs[0]
	if got.Name != "go-widgets" || got.Repos != 2 || !got.Site || !got.Docs {
		t.Errorf("go-widgets resolved to %+v, want 2 repos with site and docs", got)
	}
	if !strings.Contains(c.Summary(), "4 organisations, 5 public code repositories") {
		t.Errorf("Summary() = %q", c.Summary())
	}
	if _, ok := c.Family("nope"); ok {
		t.Error("Family(nope) reported found")
	}
}

func TestBuildRejectsVanishedOrg(t *testing.T) {
	inv := fixtureInventory()
	delete(inv, "go-gfx")
	_, err := buildFixture(t, goodClassification, inv)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("Build error = %v, want it to mention the organisation does not exist", err)
	}
}

func TestBuildRejectsEmptyClassifiedOrg(t *testing.T) {
	// An organisation that lost its last public repository must not be published
	// as "0 repos"; it belongs in the reserved list, and saying so in the error
	// is what makes the fix obvious.
	inv := fixtureInventory()
	inv["go-gfx"] = []Repo{{Name: "brand"}}
	_, err := buildFixture(t, goodClassification, inv)
	if err == nil || !strings.Contains(err.Error(), "move it to reserved") {
		t.Fatalf("Build error = %v, want it to suggest the reserved list", err)
	}
}

func TestBuildRejectsUnclassifiedOrg(t *testing.T) {
	// This is the check the whole tool exists for: a new organisation appears,
	// nobody adds it to the index, and an index that omits it silently teaches
	// the reader the capability does not exist.
	inv := fixtureInventory()
	inv["go-newthing"] = []Repo{{Name: "newthing"}}
	inv["go-alsonew"] = []Repo{{Name: "alsonew"}}
	_, err := buildFixture(t, goodClassification, inv)
	if err == nil {
		t.Fatal("Build succeeded with an unclassified organisation")
	}
	if !strings.Contains(err.Error(), "go-alsonew, go-newthing") {
		t.Errorf("error = %q, want both names in sorted order", err)
	}
}
