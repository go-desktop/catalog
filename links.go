package catalog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Not every page here is generated. The capability lookup and the standards
// page are written by hand, and so is the landing's layout — and that is
// exactly where a link outlives what it points at. A retirement sweep that
// regenerates the map does not touch them, so go-desktop's own lookup went on
// sending readers to an organisation that had been archived, with every check
// green.
//
// This closes that: the pages are read back after they are written, and any
// GitHub link that no longer resolves to something live fails the run.

// linkRe matches a GitHub organisation or repository URL in Markdown, HTML or
// prose alike. The trailing class excludes the punctuation that ends a sentence
// or closes a link, so "…/go-icons)." yields "go-icons".
var linkRe = regexp.MustCompile(`https://github\.com/([A-Za-z0-9][A-Za-z0-9._-]*)(?:/([A-Za-z0-9][A-Za-z0-9._-]*))?`)

// LinkProblem is one link that does not point at something live.
type LinkProblem struct {
	File   string
	Target string // org or org/repo, as written
	Reason string
}

func (p LinkProblem) String() string { return p.File + ": " + p.Target + " — " + p.Reason }

// CheckLinks reports the GitHub links in text that no longer resolve.
//
// An organisation this inventory does not know is left alone: it is somebody
// else's, and this tool has no opinion about it. A RETIRED one is not — those
// are ours, deliberately dropped from the map, and a page still naming one is
// the failure this exists to catch. That is why the retirement list has to be
// passed in: without it a retired organisation is indistinguishable from an
// external one, which is precisely how the archived link survived.
func (inv Inventory) CheckLinks(file string, text []byte, retired Exclusions) []LinkProblem {
	var out []LinkProblem
	seen := map[string]bool{}
	for _, m := range linkRe.FindAllStringSubmatch(string(text), -1) {
		// A dot is legal inside a name — <org>.github.io is the obvious one — so
		// the character class has to allow it, and a URL ending a sentence then
		// swallows the full stop. GitHub does not allow a name to END in a dot,
		// so trimming trailing ones is safe and is what the prose needs.
		org, repo := strings.TrimRight(m[1], "."), strings.TrimRight(m[2], ".")
		target := org
		if repo != "" {
			target = org + "/" + repo
		}
		if seen[target] {
			continue
		}
		seen[target] = true

		if retired.Excludes(org) {
			out = append(out, LinkProblem{file, target, "organisation was retired from the map"})
			continue
		}
		if !inv.Has(org) {
			continue // not ours
		}
		if repo == "" {
			continue // the organisation exists; nothing more to check
		}
		var found *Repo
		for i := range inv[org] {
			if strings.EqualFold(inv[org][i].Name, repo) {
				found = &inv[org][i]
				break
			}
		}
		switch {
		case found == nil:
			out = append(out, LinkProblem{file, target, "no such repository"})
		case found.Private:
			out = append(out, LinkProblem{file, target, "repository is private"})
		case found.Archived:
			out = append(out, LinkProblem{file, target, "repository is archived"})
		}
	}
	return out
}

// LinkReport turns problems into one error naming every one of them, sorted so
// the message is the same on every run.
func LinkReport(problems []LinkProblem) error {
	if len(problems) == 0 {
		return nil
	}
	lines := make([]string, 0, len(problems))
	for _, p := range problems {
		lines = append(lines, "  "+p.String())
	}
	sort.Strings(lines)
	return fmt.Errorf("%d dead link(s) in the hand-written pages:\n%s",
		len(problems), strings.Join(lines, "\n"))
}
