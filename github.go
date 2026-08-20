package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// DefaultBaseURL is the GitHub REST API root.
const DefaultBaseURL = "https://api.github.com"

// Client reads organisations and repositories from the GitHub REST API. It uses
// net/http directly: the two endpoints needed here are a fraction of what a
// client library would bring, and the module stays dependency-free.
type Client struct {
	BaseURL string       // defaults to DefaultBaseURL
	Token   string       // a personal access token; required, the endpoints are authenticated
	HTTP    *http.Client // defaults to a client with a timeout
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return strings.TrimSuffix(c.BaseURL, "/")
	}
	return DefaultBaseURL
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// nextLink is the "next" relation of an RFC 5988 Link header. GitHub paginates
// with it, and following the header rather than incrementing a page counter is
// what stops the walk one page past the end instead of one page short of it.
var nextLink = regexp.MustCompile(`<([^>]+)>\s*;\s*rel="next"`)

func nextPage(h http.Header) string {
	if m := nextLink.FindStringSubmatch(h.Get("Link")); m != nil {
		return m[1]
	}
	return ""
}

// get decodes one page into out and returns the URL of the next one, if any.
func (c *Client) get(ctx context.Context, u string, out any) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("request %s: %w", u, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return "", fmt.Errorf("get %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get %s: %s", u, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return "", fmt.Errorf("decode %s: %w", u, err)
	}
	return nextPage(resp.Header), nil
}

// Orgs lists the organisations the token's user belongs to.
//
// It reads /user/orgs rather than /users/{user}/orgs on purpose: the latter
// returns only public memberships, and reports nothing at all for an account
// whose memberships are private — an empty list that looks like an answer.
func (c *Client) Orgs(ctx context.Context) ([]string, error) {
	u := c.base() + "/user/orgs?per_page=100"
	var out []string
	for u != "" {
		var page []struct {
			Login string `json:"login"`
		}
		next, err := c.get(ctx, u, &page)
		if err != nil {
			return nil, err
		}
		for _, o := range page {
			out = append(out, o.Login)
		}
		u = next
	}
	return out, nil
}

// Repos lists every repository of org the token can see, public and private
// alike. Private ones are kept in the inventory and dropped at count time, so
// that a private repository is visibly excluded rather than merely absent.
func (c *Client) Repos(ctx context.Context, org string) ([]Repo, error) {
	u := fmt.Sprintf("%s/orgs/%s/repos?per_page=100&type=all", c.base(), url.PathEscape(org))
	var out []Repo
	for u != "" {
		var page []struct {
			Name     string `json:"name"`
			Private  bool   `json:"private"`
			Archived bool   `json:"archived"`
			Fork     bool   `json:"fork"`
			Language string `json:"language"`
		}
		next, err := c.get(ctx, u, &page)
		if err != nil {
			return nil, err
		}
		for _, r := range page {
			out = append(out, Repo{
				Org: org, Name: r.Name, Private: r.Private,
				Archived: r.Archived, Fork: r.Fork, Language: r.Language,
			})
		}
		u = next
	}
	return out, nil
}

// FetchInventory reads every organisation and its repositories.
//
// The walk is sequential. It is a few hundred requests against a 5000-per-hour
// budget and runs when the map is regenerated, not in a request path; fanning it
// out would trade a minute for the chance of a partial inventory, and a partial
// inventory is what silently drops an organisation from the published map.
func (c *Client) FetchInventory(ctx context.Context) (Inventory, error) {
	orgs, err := c.Orgs(ctx)
	if err != nil {
		return nil, err
	}
	inv := Inventory{}
	for _, org := range orgs {
		repos, err := c.Repos(ctx, org)
		if err != nil {
			return nil, err
		}
		inv[org] = repos
	}
	return inv, nil
}
