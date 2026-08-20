package catalog

import (
	"os"
	"testing"
)

// TestCheckedInClassification loads the classification this repository ships.
//
// Without it, a typo in families.json — a duplicated organisation, a family with
// no blurb, a missing prose section — would only be discovered by whoever next
// tried to regenerate the map, which is the worst moment to discover it. The test
// needs no network: it validates the curated half alone.
func TestCheckedInClassification(t *testing.T) {
	f, err := os.Open("families.json")
	if err != nil {
		t.Fatalf("open families.json: %v", err)
	}
	defer f.Close()
	c, err := LoadClassification(f)
	if err != nil {
		t.Fatalf("families.json is not a valid classification: %v", err)
	}
	if len(c.Families) < 2 {
		t.Errorf("only %d families; the file looks truncated", len(c.Families))
	}
	// Every organisation the map places in a family must be named once and only
	// once; LoadClassification enforces that, so this asserts the count instead.
	if got, min := len(c.Classified()), len(c.Families); got < min {
		t.Errorf("%d classified organisations across %d families", got, min)
	}
	if len(c.Reserved) == 0 {
		t.Error("no reserved names: an empty organisation listed nowhere reads as a shipped one")
	}
}
