package sanitize

import (
	"strings"
	"testing"
)

func TestContentStripsEscapeSequences(t *testing.T) {
	// A nasty payload: clear screen, set window title, hide text, BEL, DEL, CR.
	evil := "echo hi\x1b[2J\x1b]0;PWNED\x07\x1b[8mhidden\x1b[0m\r\ndone\x7f"
	got := Content(evil)

	if strings.ContainsRune(got, 0x1b) {
		t.Fatalf("ESC byte survived: %q", got)
	}
	if strings.ContainsRune(got, 0x07) {
		t.Fatalf("BEL byte survived: %q", got)
	}
	if strings.ContainsRune(got, 0x7f) {
		t.Fatalf("DEL byte survived: %q", got)
	}
	if strings.ContainsRune(got, '\r') {
		t.Fatalf("CR byte survived: %q", got)
	}
	// Newline must be preserved (legitimate multi-line snippet).
	if !strings.Contains(got, "\n") {
		t.Fatalf("newline was stripped: %q", got)
	}
	// Visible text must remain.
	for _, want := range []string{"echo hi", "hidden", "done"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got %q", want, got)
		}
	}
}

func TestContentPreservesNewlineAndTab(t *testing.T) {
	in := "line1\n\tindented\nline3"
	if got := Content(in); got != in {
		t.Fatalf("expected unchanged, got %q", got)
	}
}

func TestContentStripsC1Controls(t *testing.T) {
	// 0x9b is the 8-bit CSI; 0x9d is OSC.
	in := "a2Jb0;xc"
	got := Content(in)
	if strings.ContainsRune(got, 0x9b) || strings.ContainsRune(got, 0x9d) {
		t.Fatalf("C1 control survived: %q", got)
	}
	if got != "a2Jb0;xc" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestContentStripsUnicodeLineSeparators(t *testing.T) {
	in := "a b c"
	if got := Content(in); strings.ContainsRune(got, 0x2028) || strings.ContainsRune(got, 0x2029) {
		t.Fatalf("unicode line separator survived: %q", got)
	}
}

func TestLineCollapsesNewlines(t *testing.T) {
	in := "title\x1b[31mred\nsecond line\there"
	got := Line(in)
	if strings.ContainsRune(got, 0x1b) {
		t.Fatalf("ESC survived in Line: %q", got)
	}
	if strings.ContainsRune(got, '\n') {
		t.Fatalf("Line must not contain newlines: %q", got)
	}
	if !strings.Contains(got, "title") || !strings.Contains(got, "second line") {
		t.Fatalf("Line dropped visible text: %q", got)
	}
}

func TestEmptyInputs(t *testing.T) {
	if Content("") != "" {
		t.Fatal("Content(\"\") should be empty")
	}
	if Line("") != "" {
		t.Fatal("Line(\"\") should be empty")
	}
}
