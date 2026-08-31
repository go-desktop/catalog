package catalog

import (
	"strings"
	"testing"
)

func linkInventory() Inventory {
	return Inventory{
		"go-widgets": {
			{Org: "go-widgets", Name: "toolkit"},
			{Org: "go-widgets", Name: "secret", Private: true},
			{Org: "go-widgets", Name: "old", Archived: true},
		},
		"go-quake2": {}, // an organisation that exists and holds nothing
	}
}

func TestCheckLinks(t *testing.T) {
	inv := linkInventory()
	retired := Exclusions{"go-iconoir": true}
	text := []byte(`
Markdown: [toolkit](https://github.com/go-widgets/toolkit) and the org
[go-widgets](https://github.com/go-widgets).
HTML: <a href="https://github.com/go-widgets/old">old</a>
Prose ending a sentence: https://github.com/go-widgets/secret.
Gone: [x](https://github.com/go-widgets/never-existed)
Retired: [y](https://github.com/go-iconoir/iconoir)
Someone else's: https://github.com/golang/go
An org we know that is empty: https://github.com/go-quake2
`)
	got := inv.CheckLinks("page.md", text, retired)
	want := map[string]string{
		"go-widgets/old":           "repository is archived",
		"go-widgets/secret":        "repository is private",
		"go-widgets/never-existed": "no such repository",
		"go-iconoir/iconoir":       "organisation was retired from the map",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d problems, want %d: %v", len(got), len(want), got)
	}
	for _, p := range got {
		w, ok := want[p.Target]
		if !ok {
			t.Errorf("unexpected problem for %s: %s", p.Target, p.Reason)
			continue
		}
		if p.Reason != w {
			t.Errorf("%s: reason %q, want %q", p.Target, p.Reason, w)
		}
		if p.File != "page.md" {
			t.Errorf("%s: file %q", p.Target, p.File)
		}
	}
}

func TestCheckLinksIgnoresWhatIsNotOurs(t *testing.T) {
	// An organisation this inventory does not know belongs to somebody else, and
	// a tool that failed on it would be unusable the first time a page cited an
	// upstream project.
	inv := linkInventory()
	text := []byte("https://github.com/golang/go and https://github.com/godbus/dbus")
	if got := inv.CheckLinks("p.md", text, Exclusions{}); len(got) != 0 {
		t.Errorf("external links were reported: %v", got)
	}
}

func TestCheckLinksReportsEachTargetOnce(t *testing.T) {
	// A name repeated across a page is one problem to fix, not twenty lines of
	// the same message.
	inv := linkInventory()
	text := []byte(strings.Repeat("https://github.com/go-widgets/old\n", 20))
	if got := inv.CheckLinks("p.md", text, Exclusions{}); len(got) != 1 {
		t.Errorf("got %d problems for one repeated link, want 1", len(got))
	}
}

func TestCheckLinksMatchesRepoCaseInsensitively(t *testing.T) {
	inv := linkInventory()
	text := []byte("https://github.com/go-widgets/Toolkit")
	if got := inv.CheckLinks("p.md", text, Exclusions{}); len(got) != 0 {
		t.Errorf("a repository named in another case was reported missing: %v", got)
	}
}

func TestLinkReport(t *testing.T) {
	if err := LinkReport(nil); err != nil {
		t.Errorf("no problems must be no error, got %v", err)
	}
	// Sorted, so the same run produces the same message and a diff of two runs
	// means something.
	err := LinkReport([]LinkProblem{
		{"z.md", "a/b", "gone"},
		{"a.md", "c/d", "archived"},
	})
	if err == nil {
		t.Fatal("problems must be an error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "2 dead link(s)") {
		t.Errorf("message does not count the problems: %q", msg)
	}
	if strings.Index(msg, "a.md") > strings.Index(msg, "z.md") {
		t.Errorf("problems are not sorted:\n%s", msg)
	}
	if !strings.Contains(msg, "a/b — gone") {
		t.Errorf("message drops the reason: %q", msg)
	}
}

func TestCheckLinksTrimsSentencePunctuation(t *testing.T) {
	// A dot is legal inside a name — <org>.github.io — so the pattern must allow
	// it, which means a URL ending a sentence swallows the full stop. That made
	// "…/secret." read as a repository called "secret." and reported missing.
	inv := linkInventory()
	got := inv.CheckLinks("p.md", []byte("see https://github.com/go-widgets/secret."), Exclusions{})
	if len(got) != 1 || got[0].Target != "go-widgets/secret" {
		t.Fatalf("got %v, want one problem for go-widgets/secret", got)
	}
	if got[0].Reason != "repository is private" {
		t.Errorf("reason = %q", got[0].Reason)
	}
	// A landing repository, whose name really does carry dots, must survive.
	inv2 := Inventory{"go-desktop": {{Org: "go-desktop", Name: "go-desktop.github.io"}}}
	if p := inv2.CheckLinks("p.md", []byte("https://github.com/go-desktop/go-desktop.github.io"), Exclusions{}); len(p) != 0 {
		t.Errorf("a dotted repository name was reported: %v", p)
	}
}
