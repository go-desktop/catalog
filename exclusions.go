package catalog

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Exclusions is a set of organisation logins the map does not cover. Matching
// is case-insensitive, because a GitHub login is.
//
// The set is deliberately not part of the checked-in classification. Every
// other name this tool holds is a name it publishes — a family member, a
// reserved name, a sibling stack that is not Go — and a list of organisations
// that must not appear anywhere would be the one name list the published map
// contradicts by carrying it. It is supplied at fetch time instead, so an
// excluded organisation never enters the inventory and nothing downstream can
// name it.
//
// The reconciliation is unaffected: an organisation that is neither excluded
// nor classified still fails the build, so the map cannot quietly omit
// something that exists.
type Exclusions map[string]bool

// Excludes reports whether org is one of the excluded organisations.
func (e Exclusions) Excludes(org string) bool { return e[strings.ToLower(org)] }

// LoadExclusions reads one organisation login per line. Blank lines and lines
// starting with '#' are ignored, so that the file can record why a name is on
// it — which is the only place that reason can live.
func LoadExclusions(r io.Reader) (Exclusions, error) {
	e := Exclusions{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		e[strings.ToLower(line)] = true
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read exclusions: %w", err)
	}
	return e, nil
}
