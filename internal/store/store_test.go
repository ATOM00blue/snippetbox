package store

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ATOM00blue/snippetbox/internal/snippet"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "snippets.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func TestOpenMissingFileIsEmpty(t *testing.T) {
	s := tempStore(t)
	if s.Len() != 0 {
		t.Fatalf("expected empty store, got %d", s.Len())
	}
	if len(s.All()) != 0 {
		t.Fatalf("expected no snippets, got %d", len(s.All()))
	}
}

func TestAddGetPersists(t *testing.T) {
	s := tempStore(t)
	sn := snippet.New("Hello", "go", `fmt.Println("hi")`, []string{"demo", "go"})
	saved, err := s.Add(sn)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected generated id")
	}

	// Reopen from disk to confirm persistence.
	s2, err := Open(s.Path())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, err := s2.Get(saved.ID)
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got.Title != "Hello" || got.Language != "go" {
		t.Fatalf("unexpected snippet: %+v", got)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", got.Tags)
	}
}

func TestGetNotFound(t *testing.T) {
	s := tempStore(t)
	if _, err := s.Get("nope"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	s := tempStore(t)
	saved, _ := s.Add(snippet.New("Old", "", "body", nil))
	saved.Title = "New"
	saved.Content = "newbody"
	if err := s.Update(saved); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get(saved.ID)
	if got.Title != "New" || got.Content != "newbody" {
		t.Fatalf("update did not apply: %+v", got)
	}
	if !got.UpdatedAt.After(got.CreatedAt) && !got.UpdatedAt.Equal(got.CreatedAt) {
		t.Fatalf("updated_at not refreshed: created=%v updated=%v", got.CreatedAt, got.UpdatedAt)
	}
}

func TestUpdateMissing(t *testing.T) {
	s := tempStore(t)
	if err := s.Update(snippet.Snippet{ID: "ghost"}); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	s := tempStore(t)
	saved, _ := s.Add(snippet.New("X", "", "x", nil))
	if err := s.Delete(saved.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s.Len() != 0 {
		t.Fatalf("expected empty after delete, got %d", s.Len())
	}
	if err := s.Delete(saved.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound on second delete, got %v", err)
	}
}

func TestAllSortedNewestFirst(t *testing.T) {
	s := tempStore(t)
	a, _ := s.Add(snippet.New("A", "", "a", nil))
	_, _ = s.Add(snippet.New("B", "", "b", nil))
	// Touch A so it becomes the most recently updated.
	a.Content = "a2"
	if err := s.Update(a); err != nil {
		t.Fatalf("Update: %v", err)
	}
	all := s.All()
	if len(all) != 2 {
		t.Fatalf("expected 2, got %d", len(all))
	}
	if all[0].Title != "A" {
		t.Fatalf("expected A first (most recently updated), got %s", all[0].Title)
	}
}

func TestImportSkipsDuplicateIDs(t *testing.T) {
	s := tempStore(t)
	saved, _ := s.Add(snippet.New("Keep", "", "k", nil))

	incoming := []snippet.Snippet{
		{ID: saved.ID, Title: "Dup", Content: "dup"}, // duplicate id -> skipped
		{Title: "Fresh", Content: "fresh"},           // no id -> added
	}
	added, err := s.Import(incoming)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if added != 1 {
		t.Fatalf("expected 1 added, got %d", added)
	}
	if s.Len() != 2 {
		t.Fatalf("expected 2 total, got %d", s.Len())
	}
	// Original must be untouched.
	got, _ := s.Get(saved.ID)
	if got.Title != "Keep" {
		t.Fatalf("import overwrote existing snippet: %+v", got)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	s := tempStore(t)
	_, _ = s.Add(snippet.New("Round", "go", "trip", []string{"t"}))
	data, err := s.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	out := filepath.Join(t.TempDir(), "out.json")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	s2, err := Open(out)
	if err != nil {
		t.Fatalf("reopen marshaled: %v", err)
	}
	if s2.Len() != 1 {
		t.Fatalf("expected 1 snippet after round trip, got %d", s2.Len())
	}
}

func TestAtomicWriteNoTempLeftBehind(t *testing.T) {
	s := tempStore(t)
	_, _ = s.Add(snippet.New("A", "", "a", nil))
	dir := filepath.Dir(s.Path())
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestSavePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file mode bits are not meaningful on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "snippets.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := s.Add(snippet.New("Secret", "", "export TOKEN=abc", nil)); err != nil {
		t.Fatalf("Add: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat store: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("store file mode = %o, want 0600", got)
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat store dir: %v", err)
	}
	if got := di.Mode().Perm(); got != 0o700 {
		t.Fatalf("store dir mode = %o, want 0700", got)
	}
}

func TestAddSanitizesAndValidates(t *testing.T) {
	s := tempStore(t)

	// Escape-sequence-laden snippet must be stored sanitized.
	saved, err := s.Add(snippet.New("ti\x1b[2Jtle", "go", "echo\x1b]0;PWN\x07 hi", nil))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if strings.ContainsRune(saved.Title, 0x1b) || strings.ContainsRune(saved.Content, 0x1b) {
		t.Fatalf("ESC survived into store: %+v", saved)
	}

	// Reopen and confirm nothing nasty was persisted to disk.
	data, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	if strings.ContainsRune(string(data), 0x1b) {
		t.Fatalf("ESC byte persisted to store file")
	}

	// Empty title/content must be rejected.
	if _, err := s.Add(snippet.Snippet{Title: "", Content: "x"}); err == nil {
		t.Fatal("expected error for empty title")
	}
	if _, err := s.Add(snippet.Snippet{Title: "x", Content: "   "}); err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestImportSanitizesAndSkipsInvalid(t *testing.T) {
	s := tempStore(t)
	incoming := []snippet.Snippet{
		{Title: "Good\x1b[31m", Content: "ls -la\x1b[2J"}, // sanitized, added
		{Title: "", Content: "no title"},                  // invalid -> skipped
		{Title: "no content", Content: "   "},             // invalid -> skipped
	}
	added, err := s.Import(incoming)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if added != 1 {
		t.Fatalf("expected 1 added (2 invalid skipped), got %d", added)
	}
	for _, sn := range s.All() {
		if strings.ContainsRune(sn.Title, 0x1b) || strings.ContainsRune(sn.Content, 0x1b) {
			t.Fatalf("ESC survived import: %+v", sn)
		}
	}
}

func TestDefaultPathHonorsEnv(t *testing.T) {
	t.Setenv(EnvStore, filepath.Join("custom", "path.json"))
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if p != filepath.Join("custom", "path.json") {
		t.Fatalf("expected env override, got %q", p)
	}
}
