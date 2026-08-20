package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// Catalog is the resolved map: the curated classification with every countable
// fact filled in from the inventory. It is what the renderers read, and it holds
// no logic of its own — by the time one exists, the two inputs have already been
// reconciled.
type Catalog struct {
	GemsIntro      string
	ReadmeIntro    string
	ReadmeStandard string
	ReservedIntro  string
	ReadmeOutro    string
	Families       []ResolvedFamily
	Gems           []Gem
	Reserved       []Reserved
	NotGo          []NotGoEntry
	TotalOrgs      int
	TotalRepos     int
}

// ResolvedFamily is a family whose organisations carry their measured counts.
type ResolvedFamily struct {
	Key   string
	Title string
	Blurb string
	Orgs  []ResolvedOrg
}

// ResolvedOrg is one organisation as published.
type ResolvedOrg struct {
	Name  string
	Role  string
	Repos int
	Site  bool
	Docs  bool
}

// Gem is one of the per-gem Ruby organisations. They are found rather than
// classified: there are nearly two hundred, the organisation name states which
// gem it reimplements, and a hand-written list would be out of date the day a
// new gem is added.
type Gem struct {
	Name  string // the gem, e.g. "activerecord"
	Org   string // the organisation, e.g. "go-ruby-activerecord"
	Repos int
}

// GemPrefix is the organisation-name prefix that marks a per-gem organisation.
const GemPrefix = "go-ruby-"

// Build reconciles a classification against an inventory.
//
// Organisations under GemPrefix are collected automatically unless the
// classification names them explicitly, which is how the handful that are more
// than a gem wrapper — a widget toolkit addressable from Ruby, say — get a card
// of their own instead of a chip in the grid.
func Build(c *Classification, inv Inventory) (*Catalog, error) {
	explicit := map[string]bool{}
	for _, org := range c.Classified() {
		explicit[org] = true
	}
	for _, r := range c.Reserved {
		explicit[r.Org] = true
	}
	for _, n := range c.NotGo {
		explicit[n.Org] = true
	}

	out := &Catalog{
		GemsIntro:      c.GemsIntro,
		ReadmeIntro:    c.ReadmeIntro,
		ReadmeStandard: c.ReadmeStandard,
		ReservedIntro:  c.ReservedIntro,
		ReadmeOutro:    c.ReadmeOutro,
		Reserved:       c.Reserved,
		NotGo:          c.NotGo,
	}
	for _, f := range c.Families {
		rf := ResolvedFamily{Key: f.Key, Title: f.Title, Blurb: f.Blurb}
		for _, m := range f.Orgs {
			if !inv.Has(m.Org) {
				return nil, fmt.Errorf("%s is classified under %q but does not exist", m.Org, f.Key)
			}
			n := len(inv.Code(m.Org))
			if n == 0 {
				return nil, fmt.Errorf("%s is classified under %q but holds no public code; move it to reserved", m.Org, f.Key)
			}
			rf.Orgs = append(rf.Orgs, ResolvedOrg{
				Name:  m.Org,
				Role:  m.Role,
				Repos: n,
				Site:  inv.HasSite(m.Org),
				Docs:  inv.HasDocs(m.Org),
			})
			out.TotalOrgs++
			out.TotalRepos += n
		}
		out.Families = append(out.Families, rf)
	}

	// Gems, and the drift check in the same pass: anything holding public code
	// that is neither classified nor a gem is a gap in the map.
	var unclassified []string
	for org := range inv {
		if explicit[org] {
			continue
		}
		n := len(inv.Code(org))
		if n == 0 {
			continue // an empty organisation is reported by the reserved list, not here
		}
		if strings.HasPrefix(org, GemPrefix) {
			out.Gems = append(out.Gems, Gem{
				Name:  strings.TrimPrefix(org, GemPrefix),
				Org:   org,
				Repos: n,
			})
			out.TotalOrgs++
			out.TotalRepos += n
			continue
		}
		unclassified = append(unclassified, org)
	}
	if len(unclassified) > 0 {
		sort.Strings(unclassified)
		return nil, fmt.Errorf("holds public code but is in no family: %s", strings.Join(unclassified, ", "))
	}
	sort.Slice(out.Gems, func(i, j int) bool { return out.Gems[i].Name < out.Gems[j].Name })
	return out, nil
}

// GemRepos is the number of public code repositories across the gem
// organisations.
func (c *Catalog) GemRepos() int {
	n := 0
	for _, g := range c.Gems {
		n += g.Repos
	}
	return n
}

// Family returns the resolved family with the given key.
func (c *Catalog) Family(key string) (ResolvedFamily, bool) {
	for _, f := range c.Families {
		if f.Key == key {
			return f, true
		}
	}
	return ResolvedFamily{}, false
}
