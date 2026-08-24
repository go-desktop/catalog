// Command catalog regenerates the go-desktop ecosystem map.
//
//	catalog fetch    -out inventory.json [-token-file ~/.github-token] [-exclude-file FILE]
//	catalog check    [-inventory inventory.json] [-classification families.json]
//	catalog generate [-site ../go-desktop.github.io] [-docs ../docs] [-profile ../.github]
//
// fetch reads GitHub; check and generate read the saved inventory. Splitting
// them keeps the slow, rate-limited, network-dependent step out of the loop that
// is actually iterated on, and makes a regeneration reproducible from a file.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-desktop/catalog"
)

// osExit is the process-exit seam, so that main's behaviour is reachable from a
// test without ending the test binary.
var osExit = os.Exit

func main() { osExit(run(os.Args[1:], os.Stdout, os.Stderr)) }

const usage = `catalog regenerates the go-desktop ecosystem map.

  catalog fetch    -out FILE [-token-file FILE] [-exclude-file FILE]
                                                  read GitHub into an inventory
  catalog check                                   reconcile the two inputs
  catalog generate -site DIR -docs DIR -profile DIR
                                                  write the published files
`

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "fetch":
		err = cmdFetch(args[1:], stdout)
	case "check":
		err = cmdCheck(args[1:], stdout)
	case "generate":
		err = cmdGenerate(args[1:], stdout)
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "catalog: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "catalog: %v\n", err)
		return 1
	}
	return 0
}

// flagSet returns a set that reports errors instead of exiting the process.
func flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func cmdFetch(args []string, out io.Writer) error {
	fs := flagSet("fetch")
	dest := fs.String("out", "inventory.json", "write the inventory here")
	tokenFile := fs.String("token-file", "", "read the API token from this file")
	baseURL := fs.String("api", catalog.DefaultBaseURL, "GitHub API root")
	excludeFile := fs.String("exclude-file", "", "skip the organisations named in this file, one login per line")
	if err := fs.Parse(args); err != nil {
		return err
	}
	token := os.Getenv("GITHUB_TOKEN")
	if *tokenFile != "" {
		b, err := os.ReadFile(*tokenFile)
		if err != nil {
			return fmt.Errorf("read token: %w", err)
		}
		token = strings.TrimSpace(string(b))
	}
	if token == "" {
		return fmt.Errorf("no token: pass -token-file or set GITHUB_TOKEN")
	}
	var skip catalog.Exclusions
	if *excludeFile != "" {
		f, err := os.Open(*excludeFile)
		if err != nil {
			return fmt.Errorf("read exclusions: %w", err)
		}
		skip, err = catalog.LoadExclusions(f)
		f.Close()
		if err != nil {
			return err
		}
	}
	c := &catalog.Client{BaseURL: *baseURL, Token: token, Skip: skip}
	inv, err := c.FetchInventory(context.Background())
	if err != nil {
		return err
	}
	if err := catalog.WriteInventory(*dest, inv); err != nil {
		return err
	}
	repos := 0
	for org := range inv {
		repos += len(inv[org])
	}
	fmt.Fprintf(out, "%d organisations, %d repositories -> %s\n", len(inv), repos, *dest)
	if len(skip) > 0 {
		fmt.Fprintf(out, "%d organisation names excluded by %s\n", len(skip), *excludeFile)
	}
	return nil
}

// load reads and reconciles both inputs, which is all check does and the first
// thing generate does.
func load(inventory, classification string) (*catalog.Catalog, error) {
	f, err := os.Open(classification)
	if err != nil {
		return nil, fmt.Errorf("open classification: %w", err)
	}
	defer f.Close()
	cl, err := catalog.LoadClassification(f)
	if err != nil {
		return nil, err
	}
	inv, err := catalog.ReadInventory(inventory)
	if err != nil {
		return nil, err
	}
	return catalog.Build(cl, inv)
}

func addInputFlags(fs *flag.FlagSet) (inv, cl *string) {
	return fs.String("inventory", "inventory.json", "the saved inventory"),
		fs.String("classification", "families.json", "the curated classification")
}

func cmdCheck(args []string, out io.Writer) error {
	fs := flagSet("check")
	inv, cl := addInputFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := load(*inv, *cl)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s\n", c.Summary())
	for _, f := range c.Families {
		fmt.Fprintf(out, "  %-34s %2d orgs\n", f.Title, len(f.Orgs))
	}
	fmt.Fprintf(out, "  %-34s %2d orgs, %d repos\n", "Ruby gems", len(c.Gems), c.GemRepos())
	fmt.Fprintf(out, "  %-34s %2d\n", "reserved (no code)", len(c.Reserved))
	return nil
}

func cmdGenerate(args []string, out io.Writer) error {
	fs := flagSet("generate")
	inv, cl := addInputFlags(fs)
	site := fs.String("site", "", "the landing-page repository to write into")
	docs := fs.String("docs", "", "the documentation repository to write into")
	profile := fs.String("profile", "", "the .github repository to write the profile into")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *site == "" && *docs == "" && *profile == "" {
		return fmt.Errorf("nothing to write: pass -site, -docs, -profile, or several")
	}
	c, err := load(*inv, *cl)
	if err != nil {
		return err
	}
	written := 0
	write := func(path string, data []byte) error {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(out, "wrote %s\n", path)
		written++
		return nil
	}
	if *site != "" {
		if err := write(filepath.Join(*site, "hugo.toml"), c.HugoTOML()); err != nil {
			return err
		}
		if err := write(filepath.Join(*site, "data", "ecosystem.toml"), c.EcosystemTOML()); err != nil {
			return err
		}
	}
	if *docs != "" {
		for _, f := range c.Families {
			if err := write(filepath.Join(*docs, "docs", "families", f.Key+".md"), f.Page()); err != nil {
				return err
			}
		}
		if err := write(filepath.Join(*docs, "docs", "gems.md"), c.GemsPage()); err != nil {
			return err
		}
		if err := write(filepath.Join(*docs, "docs", "reserved.md"), c.ReservedPage()); err != nil {
			return err
		}
		if err := write(filepath.Join(*docs, "docs", "index.md"), c.DocsIndexPage()); err != nil {
			return err
		}
	}
	if *profile != "" {
		if err := write(filepath.Join(*profile, "profile", "README.md"), c.ProfileREADME()); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "%s\n%d files written\n", c.Summary(), written)
	return nil
}
