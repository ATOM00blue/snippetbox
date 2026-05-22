// Package search provides fuzzy search over snippets across title, tags,
// language and content.
package search

import (
	"sort"
	"strings"

	"github.com/ATOM00blue/snippetbox/internal/snippet"
	"github.com/sahilm/fuzzy"
)

// Result is a snippet paired with its fuzzy match score (higher is better).
type Result struct {
	Snippet snippet.Snippet
	Score   int
}

// haystackSource adapts a slice of snippets to fuzzy.Source.
type haystackSource []snippet.Snippet

func (h haystackSource) String(i int) string { return h[i].Haystack() }
func (h haystackSource) Len() int            { return len(h) }

// Search returns snippets matching query, ranked best-first. An empty query
// returns all snippets sorted by most-recently-updated.
func Search(snippets []snippet.Snippet, query string) []Result {
	query = strings.TrimSpace(query)
	if query == "" {
		out := make([]Result, len(snippets))
		ordered := make([]snippet.Snippet, len(snippets))
		copy(ordered, snippets)
		sort.SliceStable(ordered, func(i, j int) bool {
			return ordered[i].UpdatedAt.After(ordered[j].UpdatedAt)
		})
		for i, sn := range ordered {
			out[i] = Result{Snippet: sn, Score: 0}
		}
		return out
	}

	matches := fuzzy.FindFrom(query, haystackSource(snippets))
	results := make([]Result, 0, len(matches))
	for _, m := range matches {
		results = append(results, Result{
			Snippet: snippets[m.Index],
			Score:   m.Score,
		})
	}
	return results
}

// Snippets returns just the snippets from a Search, dropping scores.
func Snippets(results []Result) []snippet.Snippet {
	out := make([]snippet.Snippet, len(results))
	for i, r := range results {
		out[i] = r.Snippet
	}
	return out
}

// Best returns the single best match for query, or false if there is none.
func Best(snippets []snippet.Snippet, query string) (snippet.Snippet, bool) {
	results := Search(snippets, query)
	if len(results) == 0 {
		return snippet.Snippet{}, false
	}
	return results[0].Snippet, true
}
