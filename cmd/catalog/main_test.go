package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// classification is the smallest input Build accepts: two families, one
// reserved name, one non-Go sibling.
const classification = `{
  "gems_intro": "One organisation per gem.",
  "readme_intro": "# Map\n\n{{summary}} in {{orgs}} orgs and {{repos}} repos.",
  "readme_standards": "| a | b |\n| --- | --- |\n| CGO | 0 |",
  "reserved_intro": "Held, not built.",
  "readme_outro": "Generated.",
  "docs_index": "# Home\n\n{{orgs}} orgs, {{repos}} repos.",
  "families": [
    {"key":"desktop","title":"Desktop","blurb":"Widgets.",
     "orgs":[{"org":"go-widgets","role":"The toolkit."}]},
    {"key":"graphics","title":"Graphics","blurb":"Pixels.",
     "orgs":[{"org":"go-gfx","role":"The socle."}]}
  ],
  "reserved": [{"org":"go-quake2","for":"Later."}],
  "not_go": [{"org":"libfw","what":"C."}]
}`

const inventory = `{
  "go-widgets": [{"org":"go-widgets","name":"toolkit"},
                 {"org":"go-widgets","name":"go-widgets.github.io"},
                 {"org":"go-widgets","name":"docs"}],
  "go-gfx":     [{"org":"go-gfx","name":"gfx"}],
  "go-ruby-json":[{"org":"go-ruby-json","name":"json"}],
  "go-quake2":  [],
  "libfw":      [{"org":"libfw","name":"c-fw"}]
}`

// inputs writes the two input files into a temp dir and returns their paths.
func inputs(t *testing.T) (dir, clPath, invPath string) {
	t.Helper()
	dir = t.TempDir()
	clPath = filepath.Join(dir, "families.json")
	invPath = filepath.Join(dir, "inventory.json")
	if err := os.WriteFile(clPath, []byte(classification), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invPath, []byte(inventory), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, clPath, invPath
}

// exec runs the command and returns its exit code with both streams.
func exec(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestRunDispatch(t *testing.T) {
	if code, _, errs := exec(); code != 2 || !strings.Contains(errs, "catalog regenerates") {
		t.Errorf("no args: code=%d stderr=%q", code, errs)
	}
	for _, flag := range []string{"-h", "--help", "help"} {
		if code, out, _ := exec(flag); code != 0 || !strings.Contains(out, "catalog regenerates") {
			t.Errorf("%s: code=%d out=%q", flag, code, out)
		}
	}
	code, _, errs := exec("frobnicate")
	if code != 2 || !strings.Contains(errs, `unknown command "frobnicate"`) {
		t.Errorf("unknown command: code=%d stderr=%q", code, errs)
	}
}

func TestCheck(t *testing.T) {
	_, cl, inv := inputs(t)
	code, out, errs := exec("check", "-classification", cl, "-inventory", inv)
	if code != 0 {
		t.Fatalf("check failed: code=%d stderr=%q", code, errs)
	}
	// Two classified organisations plus one gem organisation. go-widgets holds
	// three repositories but only one of them is code — the other two are its
	// docs and landing repositories, which is exactly what must not be counted.
	if !strings.Contains(out, "3 organisations, 3 public code repositories") {
		t.Errorf("check output = %q", out)
	}
	for _, want := range []string{"Desktop", "Ruby gems", "reserved (no code)"} {
		if !strings.Contains(out, want) {
			t.Errorf("check output is missing %q", want)
		}
	}
}

func TestCheckErrors(t *testing.T) {
	_, cl, inv := inputs(t)
	dir := t.TempDir()

	if code, _, _ := exec("check", "-nosuchflag"); code != 1 {
		t.Error("an unknown flag should be an error")
	}
	if code, _, errs := exec("check", "-classification", filepath.Join(dir, "nope.json")); code != 1 ||
		!strings.Contains(errs, "open classification") {
		t.Errorf("missing classification: code=%d stderr=%q", code, errs)
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := exec("check", "-classification", bad, "-inventory", inv); code != 1 ||
		!strings.Contains(errs, "decode classification") {
		t.Errorf("bad classification: code=%d stderr=%q", code, errs)
	}
	if code, _, errs := exec("check", "-classification", cl, "-inventory", filepath.Join(dir, "nope.json")); code != 1 ||
		!strings.Contains(errs, "read inventory") {
		t.Errorf("missing inventory: code=%d stderr=%q", code, errs)
	}

	// An inventory that no longer contains a classified organisation must stop
	// the build rather than publish a page that omits it.
	drifted := filepath.Join(dir, "drifted.json")
	if err := os.WriteFile(drifted, []byte(`{"go-widgets":[{"name":"toolkit"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := exec("check", "-classification", cl, "-inventory", drifted); code != 1 ||
		!strings.Contains(errs, "does not exist") {
		t.Errorf("drifted inventory: code=%d stderr=%q", code, errs)
	}
}

func TestGenerate(t *testing.T) {
	_, cl, inv := inputs(t)
	site, docs := filepath.Join(t.TempDir(), "site"), filepath.Join(t.TempDir(), "docs")
	profile := filepath.Join(t.TempDir(), "profile")
	code, out, errs := exec("generate", "-classification", cl, "-inventory", inv,
		"-site", site, "-docs", docs, "-profile", profile)
	if code != 0 {
		t.Fatalf("generate failed: code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, "8 files written") {
		t.Errorf("generate output = %q", out)
	}
	for _, rel := range []string{
		"hugo.toml",
		filepath.Join("data", "ecosystem.toml"),
	} {
		if _, err := os.Stat(filepath.Join(site, rel)); err != nil {
			t.Errorf("site is missing %s: %v", rel, err)
		}
	}
	for _, rel := range []string{
		filepath.Join("docs", "families", "desktop.md"),
		filepath.Join("docs", "families", "graphics.md"),
		filepath.Join("docs", "gems.md"),
		filepath.Join("docs", "reserved.md"),
		filepath.Join("docs", "index.md"),
	} {
		if _, err := os.Stat(filepath.Join(docs, rel)); err != nil {
			t.Errorf("docs is missing %s: %v", rel, err)
		}
	}
	readme, err := os.ReadFile(filepath.Join(profile, "profile", "README.md"))
	if err != nil {
		t.Fatalf("profile README not written: %v", err)
	}
	// The profile repeats every count the landing page carries, which is exactly
	// why it is generated: two hand-kept copies of one number are two numbers.
	if !strings.Contains(string(readme), "3 organisations, 3 public code repositories") {
		t.Errorf("profile README did not get the substituted summary:\n%s", readme)
	}
	// Either destination alone is allowed.
	if code, _, errs := exec("generate", "-classification", cl, "-inventory", inv,
		"-site", filepath.Join(t.TempDir(), "s")); code != 0 {
		t.Errorf("site only: code=%d stderr=%q", code, errs)
	}
}

func TestGenerateErrors(t *testing.T) {
	_, cl, inv := inputs(t)
	if code, _, _ := exec("generate", "-nosuchflag"); code != 1 {
		t.Error("an unknown flag should be an error")
	}
	if code, _, errs := exec("generate", "-classification", cl, "-inventory", inv); code != 1 ||
		!strings.Contains(errs, "nothing to write") {
		t.Errorf("no destination: code=%d stderr=%q", code, errs)
	}
	if code, _, _ := exec("generate", "-classification", cl,
		"-inventory", filepath.Join(t.TempDir(), "nope.json"), "-site", t.TempDir()); code != 1 {
		t.Error("a missing inventory should be an error")
	}

	// The profile is a third destination that can fail on its own.
	blockedProfile := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blockedProfile, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := exec("generate", "-classification", cl, "-inventory", inv,
		"-profile", filepath.Join(blockedProfile, "p")); code != 1 {
		t.Error("an unwritable profile should be an error")
	}

	// A destination that cannot be created: its parent is a file.
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := exec("generate", "-classification", cl, "-inventory", inv,
		"-site", filepath.Join(blocked, "site")); code != 1 {
		t.Errorf("unwritable site: code=%d stderr=%q", code, errs)
	}
	if code, _, errs := exec("generate", "-classification", cl, "-inventory", inv,
		"-docs", filepath.Join(blocked, "docs")); code != 1 {
		t.Errorf("unwritable docs: code=%d stderr=%q", code, errs)
	}

	// A directory where a file must be written: MkdirAll succeeds, WriteFile does not.
	site := t.TempDir()
	if err := os.Mkdir(filepath.Join(site, "hugo.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := exec("generate", "-classification", cl, "-inventory", inv, "-site", site); code != 1 {
		t.Errorf("hugo.toml is a directory: code=%d stderr=%q", code, errs)
	}

	// Each write is a separate chance to fail, so a failure after a success must
	// stop the run rather than leave the site half regenerated. Here hugo.toml
	// is written and data/ecosystem.toml cannot be.
	partial := t.TempDir()
	if err := os.WriteFile(filepath.Join(partial, "data"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := exec("generate", "-classification", cl, "-inventory", inv, "-site", partial); code != 1 ||
		!strings.Contains(out, "hugo.toml") {
		t.Errorf("second site write should fail after the first succeeded: code=%d out=%q", code, out)
	}

	// The same for the documentation side: the family pages land, gems.md cannot.
	partialDocs := t.TempDir()
	if err := os.MkdirAll(filepath.Join(partialDocs, "docs", "gems.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := exec("generate", "-classification", cl, "-inventory", inv, "-docs", partialDocs); code != 1 ||
		!strings.Contains(out, "desktop.md") {
		t.Errorf("gems.md write should fail after the family pages: code=%d out=%q", code, out)
	}

	// And once more one step later: gems.md lands, reserved.md cannot.
	lastDocs := t.TempDir()
	if err := os.MkdirAll(filepath.Join(lastDocs, "docs", "reserved.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := exec("generate", "-classification", cl, "-inventory", inv, "-docs", lastDocs); code != 1 ||
		!strings.Contains(out, "gems.md") {
		t.Errorf("reserved.md write should fail after gems.md: code=%d out=%q", code, out)
	}

	// And the last of them: index.md.
	indexDocs := t.TempDir()
	if err := os.MkdirAll(filepath.Join(indexDocs, "docs", "index.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := exec("generate", "-classification", cl, "-inventory", inv, "-docs", indexDocs); code != 1 ||
		!strings.Contains(out, "reserved.md") {
		t.Errorf("index.md write should fail after reserved.md: code=%d out=%q", code, out)
	}
}

// fakeAPI serves the two endpoints fetch needs.
func fakeAPI(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"login":"go-gfx"}]`)
	})
	mux.HandleFunc("/orgs/go-gfx/repos", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"name":"gfx"},{"name":"brand"}]`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestFetch(t *testing.T) {
	dir := t.TempDir()
	api := fakeAPI(t)
	tokenFile := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenFile, []byte("  t0ken\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "inventory.json")
	code, out, errs := exec("fetch", "-api", api, "-token-file", tokenFile, "-out", dest)
	if code != 0 {
		t.Fatalf("fetch failed: code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, "1 organisations, 2 repositories") {
		t.Errorf("fetch output = %q", out)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("inventory not written: %v", err)
	}

	// The token may also come from the environment.
	t.Setenv("GITHUB_TOKEN", "env-token")
	if code, _, errs := exec("fetch", "-api", api, "-out", filepath.Join(dir, "b.json")); code != 0 {
		t.Errorf("fetch with GITHUB_TOKEN: code=%d stderr=%q", code, errs)
	}
}

func TestFetchErrors(t *testing.T) {
	dir := t.TempDir()
	api := fakeAPI(t)
	t.Setenv("GITHUB_TOKEN", "")

	if code, _, _ := exec("fetch", "-nosuchflag"); code != 1 {
		t.Error("an unknown flag should be an error")
	}
	if code, _, errs := exec("fetch", "-token-file", filepath.Join(dir, "nope")); code != 1 ||
		!strings.Contains(errs, "read token") {
		t.Errorf("missing token file: code=%d stderr=%q", code, errs)
	}
	if code, _, errs := exec("fetch", "-api", api, "-out", filepath.Join(dir, "x.json")); code != 1 ||
		!strings.Contains(errs, "no token") {
		t.Errorf("no token at all: code=%d stderr=%q", code, errs)
	}
	t.Setenv("GITHUB_TOKEN", "t0ken")
	if code, _, _ := exec("fetch", "-api", "http://\x7f", "-out", filepath.Join(dir, "x.json")); code != 1 {
		t.Error("an unreachable API should be an error")
	}
	t.Setenv("GITHUB_TOKEN", "")

	// Fetching succeeds but the inventory cannot be written.
	tokenFile := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenFile, []byte("t0ken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := exec("fetch", "-api", api, "-token-file", tokenFile,
		"-out", filepath.Join(dir, "no", "such", "x.json")); code != 1 ||
		!strings.Contains(errs, "write") {
		t.Errorf("unwritable inventory: code=%d stderr=%q", code, errs)
	}
}

// TestMain_exits covers main itself through the process-exit seam, so that the
// wiring between main, run and os.Exit is verified rather than assumed.
func TestMain_exits(t *testing.T) {
	realExit, realArgs := osExit, os.Args
	defer func() { osExit, os.Args = realExit, realArgs }()
	got := -1
	osExit = func(code int) { got = code }
	os.Args = []string{"catalog", "--help"}
	main()
	if got != 0 {
		t.Errorf("main() exited %d, want 0", got)
	}
	os.Args = []string{"catalog"}
	main()
	if got != 2 {
		t.Errorf("main() with no command exited %d, want 2", got)
	}
}
