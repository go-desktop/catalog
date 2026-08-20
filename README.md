# catalog

[![ci](https://github.com/go-desktop/catalog/actions/workflows/ci.yml/badge.svg)](https://github.com/go-desktop/catalog/actions/workflows/ci.yml)
![coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-desktop/catalog.svg)](https://pkg.go.dev/github.com/go-desktop/catalog)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)

**Generates the [go-desktop](https://go-desktop.github.io/) ecosystem map from the
live GitHub organisation list.** Pure Go, `CGO_ENABLED=0`, no dependencies.

## Why it exists

The map covers hundreds of organisations, and every one of them carries a
repository count, a landing-page link and a documentation link. Written by hand,
those numbers are wrong within a week — and a stale index is worse than no index,
because a reader concludes a capability does not exist and rebuilds it.

So the two halves are kept apart:

- **`families.json`** — the curated half. Which family an organisation belongs to,
  what its libraries are for, which names are reserved, which siblings are not Go.
  Prose, written by a person, reviewed as prose.
- **the GitHub API** — the countable half. How many public repositories hold code,
  whether a landing page and documentation are published.

Nothing countable is ever written into the curated file, and no prose is ever
inferred from the API.

## The check that matters

`Build` refuses to produce a catalogue when the two halves disagree:

| Disagreement | What it means |
| --- | --- |
| classified organisation does not exist | a name changed or an organisation was deleted |
| classified organisation holds no public code | its last library moved; it belongs in the reserved list |
| organisation holds public code but is in no family | **a new organisation nobody added to the index** |

The third is the one this tool is for. A new organisation is created long before
anyone remembers to index it, and an index that omits it silently is how a
capability gets built twice.

Organisations named `go-ruby-*` are the exception: there are nearly two hundred,
the name states which gem the organisation reimplements, so they are collected
automatically rather than listed by hand.

## Use

```sh
# read GitHub once (a few hundred requests, sequential on purpose)
catalog fetch -token-file ~/.github-token -out inventory.json

# reconcile the two inputs and print what the map contains
catalog check

# write the published files
catalog generate \
  -site    ../go-desktop.github.io \
  -docs    ../docs \
  -profile ../.github
```

`fetch` is separate from `generate` so that a regeneration is reproducible from a
file, and so the slow, rate-limited step stays out of the loop being iterated on.
The saved inventory is sorted, so a re-fetch produces a diff only where GitHub
actually changed.

## What it writes

| File | Contents |
| --- | --- |
| `<site>/hugo.toml` | one array-of-tables entry per family, with each organisation's measured counts |
| `<site>/data/ecosystem.toml` | the gem organisations, the reserved names, the gem repository total |
| `<docs>/docs/families/<key>.md` | one documentation page per family |
| `<docs>/docs/gems.md` | the alphabetical gem table |
| `<docs>/docs/reserved.md` | reserved names and the siblings outside the map |
| `<profile>/profile/README.md` | the organisation profile |

Everything else on those sites — the layout, the capability lookup, the standards
page — is hand-written and never touched.

## Standards

Pure Go, `CGO_ENABLED=0`, **zero dependencies**: the two API endpoints needed here
are a fraction of what a client library would bring. **100% statement coverage** as
a CI gate, error branches included — which is why the JSON encoder is behind an
unexported helper, and why a family renders itself instead of being looked up by a
key that cannot be missing. Validated on all six of Go's 64-bit targets.
BSD-3-Clause.
