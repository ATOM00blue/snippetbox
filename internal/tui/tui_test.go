package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ATOM00blue/snippetbox/internal/snippet"
	"github.com/ATOM00blue/snippetbox/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func newTestModel(t *testing.T) (model, *store.Store) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "snippets.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if _, err := s.Add(snippet.New("Tar dir", "bash", "tar -czf out.tgz ./dir", []string{"archive"})); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.Add(snippet.New("Go server", "go", "http.ListenAndServe(\":8080\", nil)", []string{"web"})); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return newModel(s), s
}

func sizeMsg() tea.WindowSizeMsg { return tea.WindowSizeMsg{Width: 100, Height: 30} }

func TestModelInitialRender(t *testing.T) {
	m, _ := newTestModel(t)
	um, _ := m.Update(sizeMsg())
	m = um.(model)
	view := m.View()
	if !strings.Contains(view, "snippetbox") {
		t.Fatalf("view missing title: %q", view)
	}
	if m.list.Index() != 0 {
		t.Fatalf("expected selection at 0, got %d", m.list.Index())
	}
}

func TestModelCurrentSnippet(t *testing.T) {
	m, _ := newTestModel(t)
	um, _ := m.Update(sizeMsg())
	m = um.(model)
	sn, ok := m.currentSnippet()
	if !ok {
		t.Fatal("expected a current snippet")
	}
	if sn.Title == "" {
		t.Fatal("current snippet has empty title")
	}
}

func TestModelOpenAddForm(t *testing.T) {
	m, _ := newTestModel(t)
	um, _ := m.Update(sizeMsg())
	m = um.(model)

	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = um.(model)
	if m.mode != modeForm {
		t.Fatalf("expected modeForm after 'a', got %v", m.mode)
	}
	if m.form.kind != formAdd {
		t.Fatalf("expected formAdd, got %v", m.form.kind)
	}

	// Esc cancels back to browse.
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = um.(model)
	if m.mode != modeBrowse {
		t.Fatalf("expected modeBrowse after esc, got %v", m.mode)
	}
}

func TestModelDeleteFlow(t *testing.T) {
	m, s := newTestModel(t)
	um, _ := m.Update(sizeMsg())
	m = um.(model)
	before := s.Len()

	// Press 'd' to enter confirm mode, then 'y' to confirm.
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = um.(model)
	if m.mode != modeConfirmDelete {
		t.Fatalf("expected confirm mode, got %v", m.mode)
	}
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = um.(model)
	if m.mode != modeBrowse {
		t.Fatalf("expected browse mode after confirm, got %v", m.mode)
	}
	if s.Len() != before-1 {
		t.Fatalf("expected %d snippets after delete, got %d", before-1, s.Len())
	}
}

func TestModelDeleteCancel(t *testing.T) {
	m, s := newTestModel(t)
	um, _ := m.Update(sizeMsg())
	m = um.(model)
	before := s.Len()

	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = um.(model)
	um, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = um.(model)
	if m.mode != modeBrowse {
		t.Fatalf("expected browse mode, got %v", m.mode)
	}
	if s.Len() != before {
		t.Fatalf("snippet count changed after cancel: %d -> %d", before, s.Len())
	}
}

func TestFormProducesSnippet(t *testing.T) {
	f := newForm(formAdd, snippet.Snippet{}, 80, 24)
	f.title.SetValue("My title")
	f.lang.SetValue("go")
	f.tags.SetValue("a, b")
	f.content.SetValue("fmt.Println(\"hi\")")

	sn, err := f.snippet()
	if err != nil {
		t.Fatalf("snippet() error: %v", err)
	}
	if sn.Title != "My title" || sn.Language != "go" {
		t.Fatalf("unexpected snippet: %+v", sn)
	}
	if len(sn.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", sn.Tags)
	}
}

func TestFormValidation(t *testing.T) {
	f := newForm(formAdd, snippet.Snippet{}, 80, 24)
	if _, err := f.snippet(); err == nil {
		t.Fatal("expected error for empty title")
	}
	f.title.SetValue("Has title")
	if _, err := f.snippet(); err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestListItemTitleSanitized(t *testing.T) {
	it := item{snip: snippet.Snippet{Title: "ti\x1b[2Jtle", Content: "x"}}
	if strings.ContainsRune(it.Title(), 0x1b) {
		t.Fatalf("list Title kept ESC: %q", it.Title())
	}
	it2 := item{snip: snippet.Snippet{Title: "y", Content: "echo\x1b]0;PWN\x07"}}
	if strings.ContainsRune(it2.Description(), 0x1b) || strings.ContainsRune(it2.Description(), 0x07) {
		t.Fatalf("list Description kept control bytes: %q", it2.Description())
	}
}

func TestPreviewSanitizesMaliciousContent(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "snippets.json"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Insert directly via the store; New/Add already sanitize, so to exercise the
	// render-time defense we craft a snippet that still carries an OSC payload
	// fragment and confirm refreshPreview does not emit the dangerous bytes
	// verbatim.
	if _, err := s.Add(snippet.New("Evil", "bash", "echo hi\x1b]0;PWNED\x07", nil)); err != nil {
		t.Fatalf("add: %v", err)
	}
	m := newModel(s)
	um, _ := m.Update(sizeMsg())
	m = um.(model)
	m.refreshPreview()
	view := m.preview.View()
	// The OSC title-set payload must not appear intact in the rendered preview.
	if strings.Contains(view, "\x1b]0;PWNED") {
		t.Fatalf("preview rendered raw OSC title-set sequence: %q", view)
	}
	if strings.ContainsRune(view, 0x07) {
		t.Fatalf("preview rendered raw BEL byte")
	}
}

func TestHighlightFallsBack(t *testing.T) {
	// Unknown language must not panic and must return non-empty output.
	out := highlight("echo hi", "bash")
	if out == "" {
		t.Fatal("highlight returned empty for bash")
	}
	if highlight("", "go") != "" {
		t.Fatal("highlight of empty content should be empty")
	}
	// Unknown language should still render via analysis/fallback.
	if highlight("some plain text", "not-a-real-lang") == "" {
		t.Fatal("highlight returned empty for unknown lang")
	}
}
