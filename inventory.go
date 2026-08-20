package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// The inventory is stored sorted so that a regeneration produces a diff only
// where GitHub actually changed. Go's map iteration order would otherwise make
// every fetch look like a change, which is the fastest way to teach a reviewer
// to stop reading the diff.

// writeJSON encodes v as indented JSON and writes it to path.
//
// It is separate from WriteInventory so that both of its failures are reachable
// from a test: an inventory cannot fail to encode, and a function whose error
// path cannot be exercised is a function whose error path is unverified.
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// WriteInventory saves inv with organisations and repositories in a stable
// order.
func WriteInventory(path string, inv Inventory) error {
	orgs := make([]string, 0, len(inv))
	for org := range inv {
		orgs = append(orgs, org)
	}
	sort.Strings(orgs)
	sorted := make(Inventory, len(inv))
	for _, org := range orgs {
		repos := append([]Repo(nil), inv[org]...)
		sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
		sorted[org] = repos
	}
	return writeJSON(path, sorted)
}

// ReadInventory loads an inventory saved by WriteInventory.
//
// An empty file is rejected rather than accepted as "an ecosystem of nothing":
// a truncated or half-written inventory would otherwise regenerate the map into
// a blank page without failing.
func ReadInventory(path string) (Inventory, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read inventory: %w", err)
	}
	var inv Inventory
	if err := json.Unmarshal(b, &inv); err != nil {
		return nil, fmt.Errorf("decode inventory: %w", err)
	}
	if len(inv) == 0 {
		return nil, fmt.Errorf("inventory %s is empty", path)
	}
	return inv, nil
}
