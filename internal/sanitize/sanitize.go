// Package sanitize removes terminal control sequences from untrusted text.
//
// Snippet content is attacker-controlled (imported, pasted, or synced). When
// rendered to a terminal — in the TUI preview/list or via the `show`/`list`
// subcommands — embedded ANSI/OSC escape sequences would otherwise be
// interpreted by the terminal, allowing screen clears, title rewrites, hidden
// text, and clipboard/response injection. These helpers strip the dangerous
// bytes before anything reaches the terminal.
package sanitize

import "strings"

// Content removes C0/C1 control characters and other terminal-dangerous runes
// from s while preserving the structure that legitimate snippets need: newline
// (\n) and tab (\t). Carriage returns are dropped (CRLF becomes LF). The ESC
// (0x1b) byte that begins every ANSI/OSC/DCS sequence is removed, which neuters
// the sequences without trying to parse them.
func Content(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r == '\r':
			// drop; \n is kept so CRLF collapses to LF
		case r < 0x20:
			// other C0 controls (incl. ESC 0x1b, BEL 0x07) -> drop
		case r == 0x7f:
			// DEL -> drop
		case r >= 0x80 && r <= 0x9f:
			// C1 controls (incl. 8-bit CSI/OSC) -> drop
		case r == 0x2028 || r == 0x2029:
			// Unicode line/paragraph separators -> drop
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Line is like Content but also collapses newlines and tabs into single spaces,
// producing a single safe line suitable for list rows, titles and meta fields.
func Line(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t' || r == '\r':
			b.WriteByte(' ')
		case r < 0x20:
			// other C0 controls (incl. ESC, BEL) -> drop
		case r == 0x7f:
			// DEL -> drop
		case r >= 0x80 && r <= 0x9f:
			// C1 controls -> drop
		case r == 0x2028 || r == 0x2029:
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
