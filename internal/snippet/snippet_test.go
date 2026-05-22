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

func TestNewSanitizesControlChars(t *testing.T) {
	s := New("ti\x1b[2Jtle", "g\x1bo", "echo\x1b]0;PWN\x07 hi\nsecond", []string{"ta\x1bg"})
	if strings.ContainsRune(s.Title, 0x1b) {
		t.Fatalf("title kept ESC: %q", s.Title)
	}
	if strings.ContainsRune(s.Language, 0x1b) {
		t.Fatalf("language kept ESC: %q", s.Language)
	}
	if strings.ContainsRune(s.Content, 0x1b) || strings.ContainsRune(s.Content, 0x07) {
		t.Fatalf("content kept control bytes: %q", s.Content)
	}
	// Legitimate newline must survive in content.
	if !strings.Contains(s.Content, "\n") {
		t.Fatalf("content lost newline: %q", s.Content)
	}
	for _, tag := range s.Tags {
		if strings.ContainsRune(tag, 0x1b) {
			t.Fatalf("tag kept ESC: %q", tag)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := (Snippet{Title: "x", Content: "y"}).Validate(); err != nil {
		t.Fatalf("valid snippet rejected: %v", err)
	}
	if err := (Snippet{Title: "", Content: "y"}).Validate(); err == nil {
		t.Fatal("empty title accepted")
	}
	if err := (Snippet{Title: "x", Content: " "}).Validate(); err == nil {
		t.Fatal("blank content accepted")
	}
}

func TestSanitizeCapsLength(t *testing.T) {
	long := strings.Repeat("a", MaxTitleLen+50)
	s := Snippet{Title: long, Content: "x"}.Sanitize()
	if len(s.Title) > MaxTitleLen {
		t.Fatalf("title not capped: len=%d", len(s.Title))
	}
}

func TestSummary(t *testing.T) {
	s := New("My Title", "bash", "echo hi", []string{"shell"})
	sum := s.Summary()
	if !strings.Contains(sum, "My Title") || !strings.Contains(sum, "[bash]") || !strings.Contains(sum, "#shell") {
		t.Fatalf("unexpected summary: %q", sum)
	}
}
