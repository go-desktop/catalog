package catalog

import "testing"

func TestRepoIsInfra(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{".github", true},
		{"brand", true},
		{"docs", true},
		{"benchmarks", true},
		{"go-widgets.github.io", true},
		{"renovate-runner", true},
		{"toolkit", false},
		{"documentation", false}, // not "docs"
		{"githubio", false},
	} {
		if got := (Repo{Name: tc.name}).IsInfra(); got != tc.want {
			t.Errorf("Repo{%q}.IsInfra() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestRepoIsCode(t *testing.T) {
	// An archived or forked repository still counts: a library that stopped
	// moving is still a library, and dropping it would report a capability as
	// having vanished.
	for _, tc := range []struct {
		r    Repo
		want bool
	}{
		{Repo{Name: "toolkit"}, true},
		{Repo{Name: "toolkit", Archived: true}, true},
		{Repo{Name: "toolkit", Fork: true}, true},
		{Repo{Name: "toolkit", Private: true}, false},
		{Repo{Name: "brand"}, false},
		{Repo{Name: "x.github.io"}, false},
	} {
		if got := tc.r.IsCode(); got != tc.want {
			t.Errorf("%+v.IsCode() = %v, want %v", tc.r, got, tc.want)
		}
	}
}

func testInventory() Inventory {
	return Inventory{
		"go-widgets": {
			{Org: "go-widgets", Name: "toolkit"},
			{Org: "go-widgets", Name: "painter"},
			{Org: "go-widgets", Name: "brand"},
			{Org: "go-widgets", Name: "docs"},
			{Org: "go-widgets", Name: "go-widgets.github.io"},
			{Org: "go-widgets", Name: "secret", Private: true},
		},
		"go-gfx": {
			{Org: "go-gfx", Name: "gfx"},
			{Org: "go-gfx", Name: "go-gfx.github.io"},
		},
		"empty-org":  {},
		"private-io": {{Org: "private-io", Name: "private-io.github.io", Private: true}},
		"private-doc": {
			{Org: "private-doc", Name: "code"},
			{Org: "private-doc", Name: "docs", Private: true},
		},
	}
}

func TestInventoryCode(t *testing.T) {
	inv := testInventory()
	if got := len(inv.Code("go-widgets")); got != 2 {
		t.Errorf("Code(go-widgets) = %d repos, want 2", got)
	}
	if got := inv.Code("empty-org"); got != nil {
		t.Errorf("Code(empty-org) = %v, want nil", got)
	}
	if got := inv.Code("nope"); got != nil {
		t.Errorf("Code(nope) = %v, want nil", got)
	}
}

func TestInventorySiteAndDocs(t *testing.T) {
	inv := testInventory()
	if !inv.HasSite("go-widgets") || !inv.HasDocs("go-widgets") {
		t.Error("go-widgets should have both a site and docs")
	}
	if !inv.HasSite("go-gfx") || inv.HasDocs("go-gfx") {
		t.Error("go-gfx should have a site and no docs")
	}
	// A private landing or docs repository is not published, so it must not be
	// advertised as a link the reader can follow.
	if inv.HasSite("private-io") {
		t.Error("a private .github.io must not count as a published site")
	}
	if inv.HasDocs("private-doc") {
		t.Error("a private docs repo must not count as published docs")
	}
	if inv.HasSite("empty-org") || inv.HasDocs("empty-org") {
		t.Error("an empty org publishes nothing")
	}
}

func TestInventoryHas(t *testing.T) {
	inv := testInventory()
	// Present-but-empty and absent are different answers: one is an
	// organisation holding nothing, the other is a name that does not exist.
	if !inv.Has("empty-org") {
		t.Error("Has(empty-org) = false, want true")
	}
	if inv.Has("nope") {
		t.Error("Has(nope) = true, want false")
	}
}
