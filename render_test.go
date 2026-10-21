package catalog

import (
	"strings"
	"testing"
)

func renderFixture(t *testing.T) *Catalog {
	t.Helper()
	c, err := buildFixture(t, goodClassification, fixtureInventory())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return c
}

func TestQuoteTOML(t *testing.T) {
	// A role string containing a quote, a backslash or a newline would otherwise
	// produce a hugo.toml that does not parse, and the landing page would fail
	// to build rather than render wrongly — but only after it was pushed.
	got := quoteTOML("a \"b\" c\\d\ne\tf")
	want := `"a \"b\" c\\d\ne\tf"`
	if got != want {
		t.Errorf("quoteTOML() = %s, want %s", got, want)
	}
}

func TestHugoTOML(t *testing.T) {
	out := string(renderFixture(t).HugoTOML())
	for _, want := range []string{
		`baseURL = "https://go-desktop.github.io/"`,
		"[[params.families]]",
		`key = "desktop"`,
		"[[params.families.orgs]]",
		`name = "go-widgets"`,
		"repos = 2",
		"site = true",
		"docs = true",
		`name = "go-gfx"`,
		"docs = false",
		"go-desktop/catalog",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("hugo.toml is missing %q", want)
		}
	}
	// Gem organisations belong in the data file's chip grid, not in a family card.
	if strings.Contains(out, "go-ruby-json") {
		t.Error("hugo.toml should not carry gem organisations")
	}
}

// The landing imports go-fleettools/landingkit for the shared design, and this
// function writes hugo.toml WHOLE — so the import and the brand block can only
// live here. Added to the landing by hand they would survive exactly until the
// next `catalog generate`, which is the silent kind of regression: the site
// would keep building, with its own stylesheet back and nobody told.
func TestHugoTOMLCarriesTheLandingKit(t *testing.T) {
	out := string(renderFixture(t).HugoTOML())
	for _, want := range []string{
		"[module]",
		"[[module.imports]]",
		`path = "github.com/go-fleettools/landingkit"`,
		"[params.brand]",
		// base and bright are the logo gradient's two stops. They are the
		// only brand values a reader can check against static/img/logo.svg
		// without re-deriving the rest, so they are the two pinned here.
		`base = "#1e40af"`,
		`bright = "#3b82f6"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("hugo.toml is missing %q", want)
		}
	}
	// Neutrals belong to the shared sheet. The kit removed the means to
	// override them precisely because two landings had grown surface tints
	// that drifted from the reference; re-adding one here would be invisible
	// until someone compared the rendered pages.
	for _, unwanted := range []string{"surface", "--bg", "line ="} {
		if strings.Contains(out, unwanted) {
			t.Errorf("hugo.toml carries %q: neutrals belong to the kit's sheet, not to a landing", unwanted)
		}
	}
}

func TestEcosystemTOML(t *testing.T) {
	out := string(renderFixture(t).EcosystemTOML())
	for _, want := range []string{`"json",`, `"stdlib",`, `"go-quake2",`, "gem_repos = 2"} {
		if !strings.Contains(out, want) {
			t.Errorf("ecosystem.toml is missing %q", want)
		}
	}
}

func TestFamilyPage(t *testing.T) {
	c := renderFixture(t)
	desktop, ok := c.Family("desktop")
	if !ok {
		t.Fatal("Family(desktop) not found")
	}
	out := string(desktop.Page())
	for _, want := range []string{
		"# Desktop",
		"Widgets.",
		"[`go-widgets`](https://github.com/go-widgets)",
		"[site](https://go-widgets.github.io/)",
		"[docs](https://go-widgets.github.io/docs/)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("family page is missing %q", want)
		}
	}
	// An organisation with no docs gets an em dash, not a link to a 404.
	graphics, ok := c.Family("graphics")
	if !ok {
		t.Fatal("Family(graphics) not found")
	}
	if gfx := string(graphics.Page()); !strings.Contains(gfx, "| — |") {
		t.Errorf("missing docs should render as an em dash, got:\n%s", gfx)
	}
}

func TestGemsPage(t *testing.T) {
	out := string(renderFixture(t).GemsPage())
	for _, want := range []string{
		"# Ruby gems",
		"One organisation per gem.",
		"| `json` | [`go-ruby-json`](https://github.com/go-ruby-json) |",
		"2 gem organisations, 2 public code repositories.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("gems page is missing %q", want)
		}
	}
}

func TestProfileREADME(t *testing.T) {
	out := string(renderFixture(t).ProfileREADME())
	for _, want := range []string{
		"# Map",
		// {{summary}}, {{orgs}} and {{repos}} are substituted, so a sentence
		// naming the size of the ecosystem cannot go stale.
		"4 organisations, 5 public code repositories in 4 orgs and 5 repos.",
		"## What every repository is held to",
		"| CGO | 0 |",
		"## Desktop",
		"[`go-widgets`](https://github.com/go-widgets) | 2 |",
		"## Ruby gems — one organisation each",
		"<details><summary><strong>2 gem organisations</strong>",
		"[`json`](https://github.com/go-ruby-json) · [`stdlib`](https://github.com/go-ruby-stdlib)",
		"## Held, not yet built",
		"[`go-quake2`](https://github.com/go-quake2) | Later. |",
		"## Not in this map",
		"[`example-c`](https://github.com/example-c) | C. |",
		"Generated.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("profile README is missing %q", want)
		}
	}
}

func TestReservedPage(t *testing.T) {
	out := string(renderFixture(t).ReservedPage())
	for _, want := range []string{
		"# Held, not yet built",
		"Held, not built.",
		"[`go-quake2`](https://github.com/go-quake2) | Later. |",
		"## Not in this map",
		"[`example-c`](https://github.com/example-c) | C. |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("reserved page is missing %q", want)
		}
	}
}

// With no sibling stack outside the map, neither surface carries the section at
// all: an empty "Not in this map" table says something is missing without saying
// what, which raises the question it exists to answer.
func TestNoNotGoSection(t *testing.T) {
	c := renderFixture(t)
	c.NotGo = nil
	for name, page := range map[string][]byte{
		"profile README": c.ProfileREADME(),
		"reserved page":  c.ReservedPage(),
	} {
		if strings.Contains(string(page), "Not in this map") {
			t.Errorf("%s still carries the section with nothing to put in it", name)
		}
	}
	// The section it follows must survive intact.
	if !strings.Contains(string(c.ReservedPage()), "# Held, not yet built") {
		t.Error("the reserved section went with it")
	}
}

func TestDocsIndexPage(t *testing.T) {
	out := string(renderFixture(t).DocsIndexPage())
	// {{gems}} and {{families}} are substituted too: the two counts a prose
	// sentence is most likely to state and least likely to be re-measured.
	if !strings.Contains(out, "# Home") || !strings.Contains(out, "4 orgs, 5 repos, 2 gems, 2 families.") {
		t.Errorf("docs index page = %q", out)
	}
}

// TestMkdocsSlugMatchesThePublishedAnchors pins the heading-id rule against
// the anchors the documentation site is actually serving.
//
// These are not invented examples: each left-hand side is a heading on a
// published page and each right-hand side is the id that page carries, minted
// by python-markdown before the site moved to Hugo. The two slug rules
// disagree — goldmark would write `graphics-images--media` for the first,
// with two hyphens — so an anchor somebody saved only keeps working while
// this function reproduces the old rule.
func TestMkdocsSlugMatchesThePublishedAnchors(t *testing.T) {
	for _, c := range []struct{ title, want string }{
		{"Graphics, images & media", "graphics-images-media"},
		{"Ruby gems — one organisation each", "ruby-gems-one-organisation-each"},
		{"The pure-Go ecosystem", "the-pure-go-ecosystem"},
		{"Held, not yet built", "held-not-yet-built"},
		{"How the map is organised", "how-the-map-is-organised"},
		{"Not in this map", "not-in-this-map"},
		// A non-ASCII LETTER is kept, unlike a non-ASCII symbol: the rule
		// drops punctuation, not alphabets.
		{"Café numérique", "café-numérique"},
	} {
		if got := mkdocsSlug(c.title); got != c.want {
			t.Errorf("mkdocsSlug(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}

// TestGeneratedPagesCarryFrontMatterAndAnchors checks the two things Hugo
// needs and MkDocs did not: front matter naming the page and its position, and
// an explicit id on the heading.
func TestGeneratedPagesCarryFrontMatterAndAnchors(t *testing.T) {
	c := renderFixture(t)
	for _, f := range c.Families {
		page := string(f.Page())
		if !strings.HasPrefix(page, "---\ntitle: ") {
			t.Errorf("%s page has no front matter: %.40q", f.Key, page)
		}
		if !strings.Contains(page, "{#"+mkdocsSlug(f.Title)+"}") {
			t.Errorf("%s page heading carries no anchor", f.Key)
		}
		if f.Weight < weightFirstFamily {
			t.Errorf("%s page weight %d is above the families section", f.Key, f.Weight)
		}
	}
	for name, page := range map[string][]byte{
		"gems":     c.GemsPage(),
		"reserved": c.ReservedPage(),
		"index":    c.DocsIndexPage(),
	} {
		if !strings.HasPrefix(string(page), "---\ntitle: ") {
			t.Errorf("%s page has no front matter", name)
		}
	}
}
