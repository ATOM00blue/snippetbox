// Package snippet defines the core Snippet data model shared across snippetbox.
package snippet

import (
	"crypto/rand"
	"math/big"
	"strings"
	"time"
)

// Snippet is a single stored piece of code or text.
type Snippet struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Tags      []string  `json:"tags,omitempty"`
	Language  string    `json:"language,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const idAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

// NewID returns a short, URL-safe, random identifier.
func NewID() string {
	const n = 6
	b := make([]byte, n)
	max := big.NewInt(int64(len(idAlphabet)))
	for i := range b {
		num, err := rand.Int(rand.Reader, max)
		if err != nil {
			// rand.Reader failing is effectively impossible; fall back to a
			// time-derived value so we never panic in a CLI tool.
			b[i] = idAlphabet[int(time.Now().UnixNano())%len(idAlphabet)]
			continue
		}
		b[i] = idAlphabet[num.Int64()]
	}
	return string(b)
}

// New builds a Snippet with generated ID and timestamps. Tags are normalized.
func New(title, language, content string, tags []string) Snippet {
	now := time.Now().UTC()
	return Snippet{
		ID:        NewID(),
		Title:     strings.TrimSpace(title),
		Tags:      NormalizeTags(tags),
		Language:  strings.TrimSpace(language),
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NormalizeTags trims, lowercases, de-duplicates and drops empty tags while
// preserving first-seen order.
func NormalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// ParseTags splits a comma-separated tag string into normalized tags.
func ParseTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return NormalizeTags(strings.Split(s, ","))
}

// TagString joins tags into a comma-separated string for display/editing.
func (s Snippet) TagString() string {
	return strings.Join(s.Tags, ", ")
}

// Haystack returns the combined searchable text for a snippet.
func (s Snippet) Haystack() string {
	var b strings.Builder
	b.WriteString(s.Title)
	b.WriteByte(' ')
	b.WriteString(strings.Join(s.Tags, " "))
	b.WriteByte(' ')
	b.WriteString(s.Language)
	b.WriteByte(' ')
	b.WriteString(s.Content)
	return b.String()
}

// Summary returns a one-line description of the snippet for list display.
func (s Snippet) Summary() string {
	parts := []string{s.Title}
	if s.Language != "" {
		parts = append(parts, "["+s.Language+"]")
	}
	if len(s.Tags) > 0 {
		parts = append(parts, "#"+strings.Join(s.Tags, " #"))
	}
	return strings.Join(parts, " ")
}
