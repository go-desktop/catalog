package catalog

import (
	"strings"
	"testing"
)

func mapAt(orgs, repos int) Catalog {
	return Catalog{TotalOrgs: orgs, TotalRepos: repos}
}

// ⛔⛔ THE FOUR THAT MUST NEVER FIRE. Each is a real string from the three
// published surfaces on 2026-11-10, and together they are why the obvious
// check — any integer before "organisations" — would have reported four false
// positives against two findings.
func TestTheStringsThatMustNeverBeFlagged(t *testing.T) {
	c := mapAt(324, 812)
	for _, tc := range []struct {
		name, text, why string
	}{
		{
			"a correct subset, qualified noun",
			"195 gem organisations, 195 public code repositories.",
			"a word sits between the number and the noun, so it is not a claim about the map's total",
		},
		{
			"another population: repositories holding a runner",
			"Across the fleet, 77 of the 116 repositories holding a runner have never fired.",
			`"repositories" alone is not the map's phrase`,
		},
		{
			"another population: every Renovate organisation",
			"On 2026-09-03 that landed on all 117 organisations at once.",
			"a lone organisation count is not the headline pair",
		},
		{
			"another population: Renovate again, in prose",
			"repositories across the 117 organisations that run Renovate were in that set",
			"no count of public code repositories beside it",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.CheckCounts("f", []byte(tc.text)); len(got) != 0 {
				t.Errorf("flagged %q\nreason it must not be: %s\ngot: %v", tc.text, tc.why, got)
			}
		})
	}
}

// The two that were actually stale, as they were written.
func TestTheStaleHeadlinePairIsFlagged(t *testing.T) {
	c := mapAt(324, 812)
	text := `description: "A map of one pure-Go ecosystem — 287 organisations and 564 public code repositories that build a desktop."`
	got := c.CheckCounts("docs/hugo.yaml", []byte(text))
	if len(got) != 1 {
		t.Fatalf("got %d problems, want 1: %v", len(got), got)
	}
	if got[0].Orgs != 287 || got[0].Repos != 564 {
		t.Errorf("read %d/%d, want 287/564", got[0].Orgs, got[0].Repos)
	}
	if !strings.Contains(got[0].String(), "324 / 812") {
		t.Errorf("the problem does not say what the map holds: %s", got[0].String())
	}
}

func TestACorrectPairIsSilent(t *testing.T) {
	c := mapAt(324, 812)
	for _, text := range []string{
		"**324 organisations. 812 public code repositories. No C anywhere.**",
		"**A map of one pure-Go ecosystem — 324 organisations, 812 public code repositories, no C anywhere.**",
		"A map of one pure-Go ecosystem — 324 organisations and 812 public code repositories that build a desktop.",
	} {
		if got := c.CheckCounts("f", []byte(text)); len(got) != 0 {
			t.Errorf("flagged a correct pair %q: %v", text, got)
		}
	}
}

// ⛔ A pair split by a line wrap is still one claim: the generated index page
// states its totals in a sentence that Markdown wraps, and a line-based check
// would have read two lone counts and been silent on both.
func TestAPairSplitByALineWrapIsStillOneClaim(t *testing.T) {
	c := mapAt(324, 812)
	text := "A map of one pure-Go ecosystem — 287 organisations and\n564 public code repositories, no C anywhere."
	if got := c.CheckCounts("f", []byte(text)); len(got) != 1 {
		t.Errorf("got %d problems, want 1 across the wrap: %v", len(got), got)
	}
}

// Two unrelated counts far apart in one document are not a pair. Without this
// the window would marry a Renovate census to a repository count elsewhere on
// the page and invent a finding.
func TestTwoDistantCountsAreNotAPair(t *testing.T) {
	c := mapAt(324, 812)
	text := "all 117 organisations run Renovate." +
		strings.Repeat(" filler words to push these apart.", 12) +
		"It holds 812 public code repositories."
	if got := c.CheckCounts("f", []byte(text)); len(got) != 0 {
		t.Errorf("married two distant counts into a claim: %v", got)
	}
}

// A page stating nothing is silent, which is what lets this run over every
// file without a list of exceptions.
func TestAPageWithNoClaimIsSilent(t *testing.T) {
	c := mapAt(324, 812)
	if got := c.CheckCounts("f", []byte("# A page about nothing in particular.\n")); len(got) != 0 {
		t.Errorf("flagged a page with no claim: %v", got)
	}
}

func TestCountReportIsSilentOnNothingAndNamesEveryProblem(t *testing.T) {
	if err := CountReport(nil); err != nil {
		t.Errorf("CountReport(nil) = %v, want nil", err)
	}
	err := CountReport([]CountProblem{
		{File: "b.md", Claim: `"x"`, Orgs: 1, Repos: 2, Wanted: "3 / 4"},
		{File: "a.md", Claim: `"y"`, Orgs: 5, Repos: 6, Wanted: "3 / 4"},
	})
	if err == nil {
		t.Fatal("CountReport on two problems returned nil")
	}
	msg := err.Error()
	for _, want := range []string{"2 stated total(s)", "a.md", "b.md"} {
		if !strings.Contains(msg, want) {
			t.Errorf("report is missing %q:\n%s", want, msg)
		}
	}
	// Sorted, so the same run gives the same message.
	if strings.Index(msg, "a.md") > strings.Index(msg, "b.md") {
		t.Errorf("report is not sorted:\n%s", msg)
	}
}
