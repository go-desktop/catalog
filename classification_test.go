package catalog

import (
	"strings"
	"testing"
)

const goodClassification = `{
  "gems_intro": "One organisation per gem.",
  "readme_intro": "# Map\n\n{{summary}} in {{orgs}} orgs and {{repos}} repos.",
  "readme_standards": "| a | b |\n| --- | --- |\n| CGO | 0 |",
  "reserved_intro": "Held, not built.",
  "readme_outro": "Generated.",
  "families": [
    {"key":"desktop","title":"Desktop","blurb":"Widgets.",
     "orgs":[{"org":"go-widgets","role":"The toolkit."}]},
    {"key":"graphics","title":"Graphics","blurb":"Pixels.",
     "orgs":[{"org":"go-gfx","role":"The socle."}]}
  ],
  "reserved": [{"org":"go-quake2","for":"Later."}],
  "not_go": [{"org":"example-c","what":"C."}]
}`

func TestLoadClassification(t *testing.T) {
	c, err := LoadClassification(strings.NewReader(goodClassification))
	if err != nil {
		t.Fatalf("LoadClassification: %v", err)
	}
	if got := len(c.Families); got != 2 {
		t.Fatalf("families = %d, want 2", got)
	}
	if c.GemsIntro == "" || c.ReadmeIntro == "" || c.ReadmeStandard == "" ||
		c.ReservedIntro == "" || c.ReadmeOutro == "" {
		t.Error("a prose section was dropped")
	}
	want := []string{"go-widgets", "go-gfx"}
	got := c.Classified()
	if len(got) != len(want) {
		t.Fatalf("Classified() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Classified()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadClassificationErrors(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"not json", `{`, "decode classification"},
		{"unknown field", `{"gems_intro":"x","families":[],"surprise":1}`, "decode classification"},
		{"no families", `{"gems_intro":"x","families":[]}`, "no families"},
		{"no gems intro", `{"families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}]}`, "no gems_intro"},
		{"no readme intro", `{"gems_intro":"g","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}]}`, "no readme_intro"},
		{"no standards", `{"gems_intro":"g","readme_intro":"i","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}]}`, "no readme_standards"},
		{"no reserved intro", `{"gems_intro":"g","readme_intro":"i","readme_standards":"s","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}]}`, "no reserved_intro"},
		{"no outro", `{"gems_intro":"g","readme_intro":"i","readme_standards":"s","reserved_intro":"r","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}]}`, "no readme_outro"},
		{"no key", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}]}`, "no key"},
		{"no title", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"key":"k","blurb":"B","orgs":[{"org":"o","role":"r"}]}]}`, "no title"},
		{"no blurb", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"key":"k","title":"T","orgs":[{"org":"o","role":"r"}]}]}`, "no blurb"},
		{"no orgs", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"key":"k","title":"T","blurb":"B","orgs":[]}]}`, "no organisations"},
		{"duplicate family", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[
			{"key":"k","title":"T","blurb":"B","orgs":[{"org":"a","role":"r"}]},
			{"key":"k","title":"T","blurb":"B","orgs":[{"org":"b","role":"r"}]}]}`, "appears twice"},
		{"no org name", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"role":"r"}]}]}`, "no organisation"},
		{"no role", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o"}]}]}`, "no role"},
		// Counted twice is the failure that matters: the page would merely look
		// odd, but the ecosystem totals would stop adding up.
		{"org in two families", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[
			{"key":"a","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]},
			{"key":"b","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}]}`, "in both"},
		{"reserved with no name", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}],
			"reserved":[{"for":"later"}]}`, "reserved entry with no organisation"},
		{"reserved and classified", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}],
			"reserved":[{"org":"o","for":"later"}]}`, "reserved but also in"},
		{"not-go with no name", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}],
			"not_go":[{"what":"C"}]}`, "not-Go entry with no organisation"},
		{"not-go with no what", `{"gems_intro":"x","readme_intro":"i","readme_standards":"s","reserved_intro":"r","readme_outro":"o","families":[{"key":"k","title":"T","blurb":"B","orgs":[{"org":"o","role":"r"}]}],
			"not_go":[{"org":"example-c"}]}`, "no description"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadClassification(strings.NewReader(tc.in))
			if err == nil {
				t.Fatalf("LoadClassification(%s) succeeded, want error containing %q", tc.name, tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}
