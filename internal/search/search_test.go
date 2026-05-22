package search

import (
	"testing"
	"time"

	"github.com/ATOM00blue/snippetbox/internal/snippet"
)

func fixtures() []snippet.Snippet {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return []snippet.Snippet{
		{ID: "1", Title: "Tar a directory", Tags: []string{"archive", "bash"}, Language: "bash", Content: "tar -czf out.tgz ./dir", UpdatedAt: base},
		{ID: "2", Title: "Go HTTP server", Tags: []string{"go", "web"}, Language: "go", Content: "http.ListenAndServe(\":8080\", nil)", UpdatedAt: base.Add(time.Hour)},
		{ID: "3", Title: "List files", Tags: []string{"bash"}, Language: "bash", Content: "ls -la", UpdatedAt: base.Add(2 * time.Hour)},
	}
}

func TestEmptyQueryReturnsAllNewestFirst(t *testing.T) {
	res := Search(fixtures(), "")
	if len(res) != 3 {
		t.Fatalf("expected 3 results, got %d", len(res))
	}
	if res[0].Snippet.ID != "3" {
		t.Fatalf("expected newest (id 3) first, got %s", res[0].Snippet.ID)
	}
}

func TestSearchMatchesTitle(t *testing.T) {
	res := Search(fixtures(), "tar")
	if len(res) == 0 {
		t.Fatal("expected at least one match for 'tar'")
	}
	if res[0].Snippet.ID != "1" {
		t.Fatalf("expected id 1 best for 'tar', got %s", res[0].Snippet.ID)
	}
}

func TestSearchMatchesContent(t *testing.T) {
	// "ListenAndServe" only appears in the content of snippet 2.
	res := Search(fixtures(), "ListenAndServe")
	if len(res) == 0 {
		t.Fatal("expected a content match")
	}
	if res[0].Snippet.ID != "2" {
		t.Fatalf("expected content match id 2, got %s", res[0].Snippet.ID)
	}
}

func TestSearchMatchesTag(t *testing.T) {
	res := Search(fixtures(), "archive")
	if len(res) == 0 || res[0].Snippet.ID != "1" {
		t.Fatalf("expected tag match id 1, got %+v", res)
	}
}

func TestSearchNoMatch(t *testing.T) {
	res := Search(fixtures(), "zzzznotpresent")
	if len(res) != 0 {
		t.Fatalf("expected no matches, got %d", len(res))
	}
}

func TestBest(t *testing.T) {
	sn, ok := Best(fixtures(), "http")
	if !ok {
		t.Fatal("expected a best match for 'http'")
	}
	if sn.ID != "2" {
		t.Fatalf("expected id 2, got %s", sn.ID)
	}

	if _, ok := Best(fixtures(), "qqqqq"); ok {
		t.Fatal("expected no best match")
	}
	if _, ok := Best(nil, ""); ok {
		t.Fatal("expected no best match for empty store")
	}
}

func TestSnippetsHelper(t *testing.T) {
	res := Search(fixtures(), "")
	snips := Snippets(res)
	if len(snips) != len(res) {
		t.Fatalf("length mismatch: %d vs %d", len(snips), len(res))
	}
}
