// Package snippet defines the core Snippet data model shared across snippetbox.
package snippet

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/ATOM00blue/snippetbox/internal/sanitize"
)

// Field length caps. These bound memory use from untrusted imports and keep the
// store and UI well-behaved. Content is generous (whole files are a valid use
// case) but still bounded.
const (
	MaxTitleLen    = 512
	MaxLanguageLen = 64
	MaxTagLen      = 64
	MaxTags        = 64
	MaxContentLen  = 1 << 20 // 1 MiB per snippet
)

// ErrInvalid is returned when a snippet fails validation.
var ErrInvalid = errors.New("invalid snippet")

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

// New builds a Snippet with generated ID and timestamps. All user-supplied
// fields are sanitized of terminal control sequences, normalized, and capped.
func New(title, language, content string, tags []string) Snippet {
	now := time.Now().UTC()
	s := Snippet{
		ID:        NewID(),
		Title:     title,
		Tags:      tags,
		Language:  language,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.Sanitize()
}

// NormalizeTags trims, lowercases, de-duplicates and drops empty tags while
// preserving first-seen order. Tags are sanitized of control characters and
// length-capped, and the overall count is bounded.
func NormalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(sanitize.Line(t)))
		if t == "" {
			continue
		}
		if len(t) > MaxTagLen {
			t = t[:MaxTagLen]
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) >= MaxTags {
			break
		}
	}
	return out
}

// truncate caps s to at most n bytes without splitting a UTF-8 rune.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// back up to a rune boundary
	for n > 0 && (s[n]&0xC0) == 0x80 {
		n--
	}
	return s[:n]
}

// Sanitize returns a copy of s with untrusted fields cleaned of terminal
// control sequences, normalized, and length-capped. It does not validate; use
// Validate (or Clean) to also enforce required fields.
func (s Snippet) Sanitize() Snippet {
	s.Title = truncate(sanitize.Line(s.Title), MaxTitleLen)
	s.Language = truncate(sanitize.Line(s.Language), MaxLanguageLen)
	s.Tags = NormalizeTags(s.Tags)
	s.Content = truncate(sanitize.Content(s.Content), MaxContentLen)
	return s
}

// Validate reports whether s has the required fields after sanitization.
func (s Snippet) Validate() error {
	if strings.TrimSpace(s.Title) == "" {
		return errors.Join(ErrInvalid, errors.New("title is required"))
	}
	if strings.TrimSpace(s.Content) == "" {
		return errors.Join(ErrInvalid, errors.New("content is required"))
	}
	return nil
}

// Clean sanitizes s and returns it with an error if it fails validation.
func (s Snippet) Clean() (Snippet, error) {
	s = s.Sanitize()
	if err := s.Validate(); err != nil {
		return Snippet{}, err
	}
	return s, nil
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
