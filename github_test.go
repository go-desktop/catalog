package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientDefaults(t *testing.T) {
	var c Client
	if got := c.base(); got != DefaultBaseURL {
		t.Errorf("base() = %q, want %q", got, DefaultBaseURL)
	}
	if c.http() == nil {
		t.Error("http() returned nil")
	}
	c2 := Client{BaseURL: "https://example.test/api/", HTTP: &http.Client{}}
	if got := c2.base(); got != "https://example.test/api" {
		t.Errorf("base() = %q, want the trailing slash removed", got)
	}
	if c2.http() != c2.HTTP {
		t.Error("http() ignored the supplied client")
	}
}

func TestNextPage(t *testing.T) {
	h := http.Header{}
	if got := nextPage(h); got != "" {
		t.Errorf("nextPage(no header) = %q, want empty", got)
	}
	h.Set("Link", `<https://api/x?page=2>; rel="next", <https://api/x?page=9>; rel="last"`)
	if got := nextPage(h); got != "https://api/x?page=2" {
		t.Errorf("nextPage = %q", got)
	}
	h.Set("Link", `<https://api/x?page=9>; rel="last"`)
	if got := nextPage(h); got != "" {
		t.Errorf("nextPage(no next rel) = %q, want empty", got)
	}
}

// paged serves two pages, linking the first to the second by Link header. It
// also records the Authorization header so the token can be asserted.
func paged(t *testing.T, path, page1, page2 string) (*httptest.Server, *string) {
	t.Helper()
	var auth string
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, page2)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s%s?page=2>; rel="next"`, srv.URL, path))
		fmt.Fprint(w, page1)
	})
	t.Cleanup(srv.Close)
	return srv, &auth
}

func TestClientOrgs(t *testing.T) {
	srv, auth := paged(t, "/user/orgs",
		`[{"login":"go-widgets"},{"login":"go-gfx"}]`,
		`[{"login":"go-tex"}]`)
	c := &Client{BaseURL: srv.URL, Token: "t0ken"}
	got, err := c.Orgs(context.Background())
	if err != nil {
		t.Fatalf("Orgs: %v", err)
	}
	want := []string{"go-widgets", "go-gfx", "go-tex"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Orgs() = %v, want %v (both pages, in order)", got, want)
	}
	if *auth != "Bearer t0ken" {
		t.Errorf("Authorization = %q, want a bearer token", *auth)
	}
}

func TestClientRepos(t *testing.T) {
	srv, _ := paged(t, "/orgs/go-widgets/repos",
		`[{"name":"toolkit","language":"Go"},{"name":"secret","private":true}]`,
		`[{"name":"old","archived":true,"fork":true}]`)
	c := &Client{BaseURL: srv.URL}
	got, err := c.Repos(context.Background(), "go-widgets")
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Repos() returned %d repos, want 3", len(got))
	}
	// Private repositories are kept and dropped at count time, so that an
	// exclusion is visible in the inventory rather than an absence.
	if !got[1].Private || got[1].Org != "go-widgets" {
		t.Errorf("second repo = %+v, want the private one, stamped with its org", got[1])
	}
	if !got[2].Archived || !got[2].Fork {
		t.Errorf("third repo = %+v, want archived and forked flags preserved", got[2])
	}
	if got[0].Language != "Go" {
		t.Errorf("language = %q, want Go", got[0].Language)
	}
}

func TestClientGetErrors(t *testing.T) {
	ctx := context.Background()

	// A URL that cannot even be turned into a request.
	bad := &Client{BaseURL: "http://\x7f"}
	if _, err := bad.Orgs(ctx); err == nil || !strings.Contains(err.Error(), "request") {
		t.Errorf("malformed URL error = %v", err)
	}

	// A server that is not listening.
	closed := httptest.NewServer(http.NotFoundHandler())
	url := closed.URL
	closed.Close()
	if _, err := (&Client{BaseURL: url}).Orgs(ctx); err == nil || !strings.Contains(err.Error(), "get ") {
		t.Errorf("transport error = %v", err)
	}

	// A status other than 200. Anything else would decode an error page as data.
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer fail.Close()
	if _, err := (&Client{BaseURL: fail.URL}).Orgs(ctx); err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("status error = %v", err)
	}

	// A 200 whose body is not the JSON promised.
	junk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "{")
	}))
	defer junk.Close()
	if _, err := (&Client{BaseURL: junk.URL}).Orgs(ctx); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Errorf("decode error = %v", err)
	}
	if _, err := (&Client{BaseURL: junk.URL}).Repos(ctx, "o"); err == nil {
		t.Error("Repos should fail on a malformed body")
	}
}

func TestFetchInventory(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"login":"go-gfx"},{"login":"empty"}]`)
	})
	mux.HandleFunc("/orgs/go-gfx/repos", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"name":"gfx"}]`)
	})
	mux.HandleFunc("/orgs/empty/repos", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	inv, err := (&Client{BaseURL: srv.URL}).FetchInventory(context.Background())
	if err != nil {
		t.Fatalf("FetchInventory: %v", err)
	}
	if len(inv) != 2 {
		t.Fatalf("inventory has %d orgs, want 2", len(inv))
	}
	// An organisation holding nothing must still be in the inventory, or Build
	// cannot tell "empty" from "vanished".
	if !inv.Has("empty") || len(inv["empty"]) != 0 {
		t.Errorf("empty org = %v, want present and empty", inv["empty"])
	}
}

func TestFetchInventoryErrors(t *testing.T) {
	ctx := context.Background()

	// Failing on the organisation list.
	if _, err := (&Client{BaseURL: "http://\x7f"}).FetchInventory(ctx); err == nil {
		t.Error("FetchInventory should fail when the org list fails")
	}

	// Failing on one organisation's repositories: a partial inventory must be an
	// error, because publishing one would silently drop organisations.
	mux := http.NewServeMux()
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[{"login":"go-gfx"}]`)
	})
	mux.HandleFunc("/orgs/go-gfx/repos", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	if _, err := (&Client{BaseURL: srv.URL}).FetchInventory(ctx); err == nil {
		t.Error("FetchInventory should fail when one org's repos fail")
	}
}
