// Package catalog builds the go-desktop ecosystem map from two inputs: a
// curated classification of what each organisation holds, and an inventory of
// what actually exists on GitHub.
//
// Keeping those two apart is the point. The prose — which family an
// organisation belongs to, and what its libraries are for — is human-written and
// checked in. Everything countable — how many repositories an organisation has,
// whether it publishes a landing page and documentation — is read from the API
// at generation time. A number on the published map is therefore never recalled
// or hand-typed, and the two inputs disagreeing is an error rather than a
// silently stale page:
//
//   - an organisation in the classification that no longer exists fails the build;
//   - an organisation that exists and holds code but is not classified fails it too.
//
// The second check is the one that matters in practice. A new organisation is
// created long before anyone remembers to add it to the index, and an index that
// quietly omits it is worse than no index — a reader concludes the capability
// does not exist and rebuilds it.
package catalog
