package catalog

import (
	"encoding/json"
	"fmt"
	"io"
)

// Classification is the curated half of the map: which family an organisation
// belongs to and what it holds, plus the names that are deliberately not
// counted. It is checked in and edited by hand; nothing countable lives here.
type Classification struct {
	// The prose sections of the generated surfaces. They live here rather than
	// in the renderer because they are content, and because the alternative —
	// hand-maintaining the organisation profile — is how a count on the
	// most-read page drifts away from the one on the least-read.
	//
	// ReadmeIntro may contain {{summary}}, {{orgs}} and {{repos}}.
	GemsIntro      string       `json:"gems_intro"`
	ReadmeIntro    string       `json:"readme_intro"`
	ReadmeStandard string       `json:"readme_standards"`
	ReservedIntro  string       `json:"reserved_intro"`
	ReadmeOutro    string       `json:"readme_outro"`
	Families       []Family     `json:"families"`
	Reserved       []Reserved   `json:"reserved"`
	NotGo          []NotGoEntry `json:"not_go"`
}

// Family is one section of the map.
type Family struct {
	Key   string        `json:"key"`
	Title string        `json:"title"`
	Blurb string        `json:"blurb"`
	Orgs  []MemberEntry `json:"orgs"`
}

// MemberEntry places one organisation in a family and says what it holds.
type MemberEntry struct {
	Org  string `json:"org"`
	Role string `json:"role"`
}

// Reserved is an organisation name held but holding no code. Listing these is
// what stops a plausible name from being read as a shipped capability.
type Reserved struct {
	Org string `json:"org"`
	For string `json:"for"`
}

// NotGoEntry is a sibling organisation deliberately outside the Go map.
type NotGoEntry struct {
	Org  string `json:"org"`
	What string `json:"what"`
}

// LoadClassification reads a classification from JSON and checks it is
// self-consistent: every field that the templates dereference is present, and
// no organisation is placed in two families at once.
//
// A duplicate is worth failing on rather than tolerating. The visible symptom
// would be an organisation appearing twice on the page, but the invisible one is
// that it is also counted twice, and the ecosystem totals stop adding up.
func LoadClassification(r io.Reader) (*Classification, error) {
	var c Classification
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("decode classification: %w", err)
	}
	if len(c.Families) == 0 {
		return nil, fmt.Errorf("classification has no families")
	}
	for _, f := range []struct {
		name, value string
	}{
		{"gems_intro", c.GemsIntro},
		{"readme_intro", c.ReadmeIntro},
		{"readme_standards", c.ReadmeStandard},
		{"reserved_intro", c.ReservedIntro},
		{"readme_outro", c.ReadmeOutro},
	} {
		if f.value == "" {
			return nil, fmt.Errorf("classification has no %s", f.name)
		}
	}
	seenFamily := map[string]bool{}
	seenOrg := map[string]string{}
	for _, f := range c.Families {
		switch {
		case f.Key == "":
			return nil, fmt.Errorf("family with no key")
		case f.Title == "":
			return nil, fmt.Errorf("family %q has no title", f.Key)
		case f.Blurb == "":
			return nil, fmt.Errorf("family %q has no blurb", f.Key)
		case len(f.Orgs) == 0:
			return nil, fmt.Errorf("family %q has no organisations", f.Key)
		case seenFamily[f.Key]:
			return nil, fmt.Errorf("family %q appears twice", f.Key)
		}
		seenFamily[f.Key] = true
		for _, m := range f.Orgs {
			switch {
			case m.Org == "":
				return nil, fmt.Errorf("family %q has an entry with no organisation", f.Key)
			case m.Role == "":
				return nil, fmt.Errorf("%s has no role", m.Org)
			}
			if prev, dup := seenOrg[m.Org]; dup {
				return nil, fmt.Errorf("%s is in both %q and %q", m.Org, prev, f.Key)
			}
			seenOrg[m.Org] = f.Key
		}
	}
	for _, r := range c.Reserved {
		if r.Org == "" {
			return nil, fmt.Errorf("reserved entry with no organisation")
		}
		if prev, dup := seenOrg[r.Org]; dup {
			return nil, fmt.Errorf("%s is reserved but also in %q", r.Org, prev)
		}
		seenOrg[r.Org] = "reserved"
	}
	for _, n := range c.NotGo {
		if n.Org == "" {
			return nil, fmt.Errorf("not-Go entry with no organisation")
		}
		if n.What == "" {
			return nil, fmt.Errorf("%s has no description", n.Org)
		}
	}
	return &c, nil
}

// Classified returns every organisation the classification places in a family,
// which is the set that must exist and hold code.
func (c *Classification) Classified() []string {
	var out []string
	for _, f := range c.Families {
		for _, m := range f.Orgs {
			out = append(out, m.Org)
		}
	}
	return out
}
