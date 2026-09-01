package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// Drift is what GitHub holds now minus what the published map was generated
// from. It exists because reconciliation alone does not notice the commonest
// way the map goes stale.
//
// Build refuses an organisation it cannot place, so a whole new organisation is
// caught the moment anyone runs check. A repository landing in an organisation
// the map already names is not: every rule still holds, nothing fails, and the
// only thing wrong is a number on a published page. On 2026-09-01 five
// repositories had appeared since the last regeneration and exactly one of them
// -- the one that brought a new organisation with it -- would have been caught
// that way. The other four had been invisible for ten days.
//
// So the drift a scheduled job looks for is a comparison against the inventory
// the pages were actually generated from, not a re-run of the rules.
type Drift struct {
	NewOrgs  []string // organisations GitHub has and the saved inventory does not
	GoneOrgs []string // organisations the saved inventory has and GitHub does not
	// Repositories are compared only for organisations both sides know, and
	// only those that count towards a published number. A vanished
	// organisation's repositories are not listed again underneath it: the
	// organisation is the finding, and repeating its contents buries it.
	NewRepos  []string // "org/name", counted by the map, newly present
	GoneRepos []string // "org/name", counted by the map, no longer present
	// The landing page and the documentation site are not code and never
	// counted, but the map prints a link to each and a dash where there is
	// none. An organisation that publishes one afterwards leaves a dash on a
	// page that is now wrong, and no count moves to say so -- which is exactly
	// the silence this type exists to break.
	NewPages  []string // "org/site" or "org/docs", published since
	GonePages []string // "org/site" or "org/docs", published no longer
}

// Empty reports whether the two inventories describe the same published map.
func (d Drift) Empty() bool {
	return len(d.NewOrgs) == 0 && len(d.GoneOrgs) == 0 &&
		len(d.NewRepos) == 0 && len(d.GoneRepos) == 0 &&
		len(d.NewPages) == 0 && len(d.GonePages) == 0
}

// CompareInventories reports how now differs from was.
//
// "Counted by the map" is IsCode, not every repository: a private repository or
// an organisation's docs site moving in or out changes nothing a reader sees,
// and a drift report that fires on those is one nobody reads.
func CompareInventories(was, now Inventory) Drift {
	var d Drift
	for org := range now {
		if !was.Has(org) {
			d.NewOrgs = append(d.NewOrgs, org)
		}
	}
	for org := range was {
		if !now.Has(org) {
			d.GoneOrgs = append(d.GoneOrgs, org)
		}
	}
	for org := range now {
		if !was.Has(org) {
			continue
		}
		before, after := codeNames(was, org), codeNames(now, org)
		for name := range after {
			if !before[name] {
				d.NewRepos = append(d.NewRepos, org+"/"+name)
			}
		}
		for name := range before {
			if !after[name] {
				d.GoneRepos = append(d.GoneRepos, org+"/"+name)
			}
		}
		for _, s := range []struct {
			kind string
			had  bool
			has  bool
		}{
			{"site", was.HasSite(org), now.HasSite(org)},
			{"docs", was.HasDocs(org), now.HasDocs(org)},
		} {
			switch {
			case !s.had && s.has:
				d.NewPages = append(d.NewPages, org+"/"+s.kind)
			case s.had && !s.has:
				d.GonePages = append(d.GonePages, org+"/"+s.kind)
			}
		}
	}
	for _, s := range []*[]string{&d.NewOrgs, &d.GoneOrgs, &d.NewRepos, &d.GoneRepos, &d.NewPages, &d.GonePages} {
		sort.Strings(*s)
	}
	return d
}

// codeNames is the set of an organisation's repositories that count towards a
// published number.
func codeNames(inv Inventory, org string) map[string]bool {
	out := make(map[string]bool)
	for _, r := range inv.Code(org) {
		out[r.Name] = true
	}
	return out
}

// Report renders the drift for a person to read, one finding per line, or a
// single line saying there is none.
func (d Drift) Report() string {
	if d.Empty() {
		return "no drift: GitHub matches the inventory the map was generated from\n"
	}
	var b strings.Builder
	section := func(title string, names []string, mark rune) {
		if len(names) == 0 {
			return
		}
		fmt.Fprintf(&b, "%s (%d):\n", title, len(names))
		for _, n := range names {
			fmt.Fprintf(&b, "  %c %s\n", mark, n)
		}
	}
	section("organisations not on the map", d.NewOrgs, '+')
	section("organisations the map names that GitHub no longer has", d.GoneOrgs, '-')
	section("repositories the map does not count", d.NewRepos, '+')
	section("repositories the map counts that are gone", d.GoneRepos, '-')
	section("pages published since, which the map still shows as absent", d.NewPages, '+')
	section("pages the map links to that are published no longer", d.GonePages, '-')
	return b.String()
}
