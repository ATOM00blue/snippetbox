// Package store handles persistence of snippets to a JSON file on disk.
//
// The store is a single JSON document. Writes are atomic (temp file + rename)
// so a crash mid-write cannot corrupt the user's snippets.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ATOM00blue/snippetbox/internal/snippet"
)

// Version is the on-disk schema version.
const Version = 1

// EnvStore is the environment variable used to override the store path.
const EnvStore = "SNIPPETBOX_STORE"

// ErrNotFound is returned when a snippet cannot be located.
var ErrNotFound = errors.New("snippet not found")

// document is the on-disk representation of the store.
type document struct {
	Version  int               `json:"version"`
	Snippets []snippet.Snippet `json:"snippets"`
}

// Store is an in-memory view of the snippet collection backed by a file.
type Store struct {
	path     string
	snippets []snippet.Snippet
}

// DefaultPath returns the platform-appropriate default store path, honoring the
// SNIPPETBOX_STORE environment variable when set.
func DefaultPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv(EnvStore)); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate config dir: %w", err)
	}
	return filepath.Join(dir, "snippetbox", "snippets.json"), nil
}

// Open loads the store at path. A path of "" uses DefaultPath. A missing file
// is treated as an empty store (it will be created on the first Save).
func Open(path string) (*Store, error) {
	if path == "" {
		var err error
		if path, err = DefaultPath(); err != nil {
			return nil, err
		}
	}
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, fmt.Errorf("read store: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s, nil
	}
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse store %q: %w", path, err)
	}
	s.snippets = doc.Snippets
	return s, nil
}

// Path returns the file path backing the store.
func (s *Store) Path() string { return s.path }

// All returns a copy of all snippets, newest-updated first.
func (s *Store) All() []snippet.Snippet {
	out := make([]snippet.Snippet, len(s.snippets))
	copy(out, s.snippets)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

// Len returns the number of snippets in the store.
func (s *Store) Len() int { return len(s.snippets) }

// Get returns the snippet with the given id.
func (s *Store) Get(id string) (snippet.Snippet, error) {
	for _, sn := range s.snippets {
		if sn.ID == id {
			return sn, nil
		}
	}
	return snippet.Snippet{}, ErrNotFound
}

// Add inserts a new snippet and persists the store. If the snippet has no ID,
// one is generated. Timestamps are set when zero.
func (s *Store) Add(sn snippet.Snippet) (snippet.Snippet, error) {
	if sn.ID == "" {
		sn.ID = snippet.NewID()
	}
	// Guarantee a unique ID even in the unlikely event of a collision.
	for s.idExists(sn.ID) {
		sn.ID = snippet.NewID()
	}
	now := time.Now().UTC()
	if sn.CreatedAt.IsZero() {
		sn.CreatedAt = now
	}
	if sn.UpdatedAt.IsZero() {
		sn.UpdatedAt = now
	}
	sn.Tags = snippet.NormalizeTags(sn.Tags)
	s.snippets = append(s.snippets, sn)
	if err := s.Save(); err != nil {
		return snippet.Snippet{}, err
	}
	return sn, nil
}

// Update replaces the snippet matching sn.ID and persists the store.
func (s *Store) Update(sn snippet.Snippet) error {
	for i := range s.snippets {
		if s.snippets[i].ID == sn.ID {
			sn.CreatedAt = s.snippets[i].CreatedAt
			sn.UpdatedAt = time.Now().UTC()
			sn.Tags = snippet.NormalizeTags(sn.Tags)
			s.snippets[i] = sn
			return s.Save()
		}
	}
	return ErrNotFound
}

// Delete removes the snippet with the given id and persists the store.
func (s *Store) Delete(id string) error {
	for i := range s.snippets {
		if s.snippets[i].ID == id {
			s.snippets = append(s.snippets[:i], s.snippets[i+1:]...)
			return s.Save()
		}
	}
	return ErrNotFound
}

// Import merges snippets from others into the store. Snippets whose ID already
// exists are skipped unless their ID is empty (then a new one is assigned).
// Returns the number of snippets added.
func (s *Store) Import(others []snippet.Snippet) (int, error) {
	added := 0
	for _, sn := range others {
		if sn.ID != "" && s.idExists(sn.ID) {
			continue
		}
		if sn.ID == "" {
			sn.ID = snippet.NewID()
		}
		for s.idExists(sn.ID) {
			sn.ID = snippet.NewID()
		}
		now := time.Now().UTC()
		if sn.CreatedAt.IsZero() {
			sn.CreatedAt = now
		}
		if sn.UpdatedAt.IsZero() {
			sn.UpdatedAt = now
		}
		sn.Tags = snippet.NormalizeTags(sn.Tags)
		s.snippets = append(s.snippets, sn)
		added++
	}
	if added > 0 {
		if err := s.Save(); err != nil {
			return 0, err
		}
	}
	return added, nil
}

// Marshal returns the store serialized as indented JSON (used by export).
func (s *Store) Marshal() ([]byte, error) {
	doc := document{Version: Version, Snippets: s.snippets}
	return json.MarshalIndent(doc, "", "  ")
}

// Save writes the store to disk atomically.
func (s *Store) Save() error {
	data, err := s.Marshal()
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create store dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".snippets-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if anything below fails before the rename.
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace store file: %w", err)
	}
	return nil
}

func (s *Store) idExists(id string) bool {
	for _, sn := range s.snippets {
		if sn.ID == id {
			return true
		}
	}
	return false
}
