package snippet

import (
	"strings"
	"testing"
)

func TestNewIDUniqueAndShaped(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewID()
		if len(id) != 6 {
			t.Fatalf("expected 6-char id, got %q", id)
		}
		for _, r := range id {
			if !strings.ContainsRune(idAlphabet, r) {
				t.Fatalf("id %q has out-of-alphabet rune %q", id, r)
			}
		}
		if seen[id] {
			t.Fatalf("duplicate id generated: %q", id)
		}
		seen[id] = true
	}
}

func TestNewNormalizes(t *testing.T) {
	s := New("  Hello  ", "  Go ", "body", []string{"A", "a", " B ", "", "b"})
	if s.Title != "Hello" {
		t.Fatalf("title not trimmed: %q", s.Title)
	}
	if s.Language != "Go" {
		t.Fatalf("language not trimmed: %q", s.Language)
	}
	if len(s.Tags) != 2 || s.Tags[0] != "a" || s.Tags[1] != "b" {
		t.Fatalf("tags not normalized/deduped: %v", s.Tags)
	}
	if s.ID == "" || s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() {
		t.Fatalf("missing id/timestamps: %+v", s)
	}
}

func TestParseTags(t *testing.T) {
	got := ParseTags("go, Web , go,,api")
	want := []string{"go", "web", "api"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if ParseTags("   ") != nil {
		t.Fatal("expected nil for blank tag string")
	}
}

func TestTagString(t *testing.T) {
	s := Snippet{Tags: []string{"a", "b", "c"}}
	if s.TagString() != "a, b, c" {
		t.Fatalf("unexpected: %q", s.TagString())
	}
}

func TestHaystackIncludesAllFields(t *testing.T) {
	s := New("Title", "go", "content body", []string{"tag1"})
	h := s.Haystack()
	for _, want := range []string{"Title", "go", "content body", "tag1"} {
		if !strings.Contains(h, want) {
			t.Fatalf("haystack %q missing %q", h, want)
		}
	}
}

func TestSummary(t *testing.T) {
	s := New("My Title", "bash", "echo hi", []string{"shell"})
	sum := s.Summary()
	if !strings.Contains(sum, "My Title") || !strings.Contains(sum, "[bash]") || !strings.Contains(sum, "#shell") {
		t.Fatalf("unexpected summary: %q", sum)
	}
}
