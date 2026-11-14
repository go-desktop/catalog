package catalog

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The read-back added in #4 fails on a dead link in a hand-written page. It
// cannot see a stale NUMBER, and a count has no 404.
//
// Measured 2026-11-10: go-desktop/docs' hugo.yaml said "287 organisations and
// 564 public code repositories" while content/en/_index.md, generated from the
// same inventory, said 324 and 812 on the same page. The generator writes
// content/en/ and never touches hugo.yaml, so the number went 564 -> 618 ->
// 755 -> 812 without it. That string is <meta name="description">, so the
// stale pair was the one search results and link previews showed and the
// correct one sat in the body nobody quotes.
//
// ⛔ THE OBVIOUS CHECK IS WRONG, AND THAT IS WHY THIS ONE IS NARROW.
// Grepping for an integer before "organisations" over the three published
// surfaces returns nine hits: two stale, three correct-and-generated, and FOUR
// that must never fire —
//
//	195 gem organisations, 195 public code repositories   (a correct SUBSET)
//	77 of the 116 repositories holding a runner           (another population)
//	all 117 organisations                                 (Renovate's, not the map's)
//	the 117 organisations that run Renovate               (likewise)
//
// Four false positives against two findings is a guard that gets switched off
// inside a week. So two rules do the separating, and neither needs a list of
// which files are hand-written — a list that would itself go stale:
//
//  1. THE HEADLINE PAIR. A count of organisations AND a count of public code
//     repositories near each other is the map's own headline shape. A lone
//     integer before "organisations" is not, which is what keeps Renovate's
//     counts and a runner census out. A lone count in a comment is better
//     deleted than asserted, and was.
//  2. AN UNQUALIFIED NOUN. "195 gem organisations" puts a word between the
//     number and the noun, so it is not a claim about the map's total. The
//     pattern requires the number to sit immediately before the noun, which
//     expresses that without a special case.

// orgsClaim matches a count of organisations with NOTHING between the number
// and the noun: "324 organisations", never "195 gem organisations".
var orgsClaim = regexp.MustCompile(`(\d+)\s+organisations\b`)

// reposClaim matches the map's own phrase for the other half. "repositories"
// alone is deliberately NOT enough: it is how a runner census and a Renovate
// report count a different population.
var reposClaim = regexp.MustCompile(`(\d+)\s+public code repositories\b`)

// pairWindow is how far apart the two halves may sit and still be one claim.
// It is generous enough to survive a wrapped Markdown paragraph and tight
// enough that two unrelated counts in one document are not read as a pair.
const pairWindow = 160

// CountProblem is one stated pair of totals that the map no longer holds.
type CountProblem struct {
	File   string
	Claim  string
	Orgs   int
	Repos  int
	Wanted string
}

func (p CountProblem) String() string {
	return fmt.Sprintf("%s: says %d organisations / %d public code repositories, map holds %s — %s",
		p.File, p.Orgs, p.Repos, p.Wanted, p.Claim)
}

// CheckCounts reports every headline pair in text that disagrees with the
// map's own totals.
//
// A file stating no pair is silent here, which is the common case and the
// reason this can run over every page without a list of exceptions.
func (c Catalog) CheckCounts(file string, text []byte) []CountProblem {
	// Collapse line breaks so a pair split by a wrap is still one claim; the
	// offsets below are into this flattened copy, used only for the window and
	// the quoted context.
	flat := strings.Join(strings.Fields(string(text)), " ")

	var out []CountProblem
	for _, rm := range reposClaim.FindAllStringSubmatchIndex(flat, -1) {
		repos, err := strconv.Atoi(flat[rm[2]:rm[3]])
		if err != nil {
			continue
		}
		om, orgs, ok := nearestOrgs(flat, rm[0])
		if !ok {
			continue // a repository count with no organisation count beside it
		}
		if orgs == c.TotalOrgs && repos == c.TotalRepos {
			continue
		}
		lo, hi := om, rm[1]
		if lo > hi {
			lo, hi = rm[0], om
		}
		out = append(out, CountProblem{
			File:   file,
			Claim:  `"` + strings.TrimSpace(flat[lo:hi]) + `"`,
			Orgs:   orgs,
			Repos:  repos,
			Wanted: fmt.Sprintf("%d / %d", c.TotalOrgs, c.TotalRepos),
		})
	}
	return out
}

// nearestOrgs finds the organisation count closest to at, within pairWindow,
// and returns where it starts.
func nearestOrgs(flat string, at int) (start, orgs int, ok bool) {
	best := -1
	for _, m := range orgsClaim.FindAllStringSubmatchIndex(flat, -1) {
		d := m[0] - at
		if d < 0 {
			d = -d
		}
		if d > pairWindow {
			continue
		}
		if best < 0 || d < best {
			n, err := strconv.Atoi(flat[m[2]:m[3]])
			if err != nil {
				continue
			}
			best, start, orgs, ok = d, m[0], n, true
		}
	}
	return start, orgs, ok
}

// CountReport turns problems into one error naming every one of them, sorted
// so the message is the same on every run.
func CountReport(problems []CountProblem) error {
	if len(problems) == 0 {
		return nil
	}
	lines := make([]string, 0, len(problems))
	for _, p := range problems {
		lines = append(lines, "  "+p.String())
	}
	sort.Strings(lines)
	return fmt.Errorf("%d stated total(s) disagree with the map:\n%s\n"+
		"A generated page cannot drift; a hand-written one has nothing watching it. "+
		"Either update the number or stop stating one.",
		len(problems), strings.Join(lines, "\n"))
}
