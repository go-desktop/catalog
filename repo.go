package catalog

import "strings"

// Repo is the part of a GitHub repository this package needs. It is the shape
// stored in an inventory file, so the field names are the JSON keys too.
type Repo struct {
	Org      string `json:"org"`
	Name     string `json:"name"`
	Private  bool   `json:"private"`
	Archived bool   `json:"archived"`
	Fork     bool   `json:"fork"`
	Language string `json:"language,omitempty"`
}

// infraNames are the repositories every organisation carries for publishing
// rather than for code. Counting them would report a one-library organisation as
// holding five.
var infraNames = map[string]bool{
	".github":    true,
	"brand":      true,
	"docs":       true,
	"benchmarks": true,
	// A self-hosted Renovate runner is a scheduled job's configuration — a few
	// kilobytes of JavaScript that watches other repositories. It is the same
	// category as docs: per-organisation plumbing nobody imports. Counting it
	// would report an organisation as holding one more library than it does.
	"renovate-runner": true,
}

// IsInfra reports whether r is one of its organisation's publishing
// repositories rather than one holding code.
//
// The landing page is matched by suffix rather than by exact name: it is
// conventionally <org>.github.io, but an organisation whose name already ends in
// .github.io would otherwise have its only repository discounted.
func (r Repo) IsInfra() bool {
	if infraNames[r.Name] {
		return true
	}
	return strings.HasSuffix(r.Name, ".github.io")
}

// IsCode reports whether r counts towards an organisation's published
// repository count: public, and not a publishing repository.
//
// Archived and forked repositories still count. An archived library is a real
// one that has stopped moving, and hiding it would make the map claim a
// capability disappeared when it merely stopped changing.
func (r Repo) IsCode() bool { return !r.Private && !r.IsInfra() }

// Inventory is what exists on GitHub, keyed by organisation login. A nil or
// missing entry and an empty one mean different things: absent is "no such
// organisation", empty is "an organisation holding nothing".
type Inventory map[string][]Repo

// Code returns the repositories of org that count towards its published count,
// in the order the inventory holds them.
func (inv Inventory) Code(org string) []Repo {
	var out []Repo
	for _, r := range inv[org] {
		if r.IsCode() {
			out = append(out, r)
		}
	}
	return out
}

// HasSite reports whether org publishes a public landing page repository.
func (inv Inventory) HasSite(org string) bool {
	for _, r := range inv[org] {
		if !r.Private && strings.HasSuffix(r.Name, ".github.io") {
			return true
		}
	}
	return false
}

// HasDocs reports whether org publishes a public documentation repository.
func (inv Inventory) HasDocs(org string) bool {
	for _, r := range inv[org] {
		if !r.Private && r.Name == "docs" {
			return true
		}
	}
	return false
}

// Has reports whether the inventory knows org at all, which is what
// distinguishes an empty organisation from a vanished one.
func (inv Inventory) Has(org string) bool {
	_, ok := inv[org]
	return ok
}
