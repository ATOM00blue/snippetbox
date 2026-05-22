package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runCLI invokes run with the given args against an isolated store, returning
// exit code, stdout and stderr.
func runCLI(t *testing.T, storePath, stdin string, args ...string) (int, string, string) {
	t.Helper()
	full := append([]string{"--store", storePath}, args...)
	var out, errb bytes.Buffer
	code := run(full, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func newStorePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "snippets.json")
}

func TestAddAndFind(t *testing.T) {
	sp := newStorePath(t)

	code, out, errb := runCLI(t, sp, "", "add", "-t", "Tar dir", "-l", "bash", "-T", "archive,shell", "-c", "tar -czf out.tgz ./dir")
	if code != 0 {
		t.Fatalf("add failed code=%d err=%s", code, errb)
	}
	if !strings.Contains(out, "added") || !strings.Contains(out, "Tar dir") {
		t.Fatalf("unexpected add output: %q", out)
	}

	code, out, _ = runCLI(t, sp, "", "find", "tar")
	if code != 0 {
		t.Fatalf("find failed code=%d", code)
	}
	if !strings.Contains(out, "Tar dir") {
		t.Fatalf("find did not return snippet: %q", out)
	}
}

func TestAddFromStdin(t *testing.T) {
	sp := newStorePath(t)
	code, _, errb := runCLI(t, sp, "package main\n", "add", "-t", "Go main", "-l", "go", "--stdin")
	if code != 0 {
		t.Fatalf("add --stdin failed: %s", errb)
	}
	code, out, _ := runCLI(t, sp, "", "show", "go main")
	if code != 0 {
		t.Fatalf("show failed code=%d", code)
	}
	if !strings.Contains(out, "package main") {
		t.Fatalf("show returned wrong content: %q", out)
	}
}

func TestAddRequiresTitle(t *testing.T) {
	sp := newStorePath(t)
	code, _, errb := runCLI(t, sp, "", "add", "-c", "body")
	if code == 0 {
		t.Fatal("expected nonzero exit when title missing")
	}
	if !strings.Contains(errb, "title") {
		t.Fatalf("expected title error, got %q", errb)
	}
}

func TestAddRequiresContent(t *testing.T) {
	sp := newStorePath(t)
	code, _, errb := runCLI(t, sp, "", "add", "-t", "No body")
	if code == 0 {
		t.Fatal("expected nonzero exit when content missing")
	}
	if !strings.Contains(errb, "content") {
		t.Fatalf("expected content error, got %q", errb)
	}
}

func TestListJSON(t *testing.T) {
	sp := newStorePath(t)
	runCLI(t, sp, "", "add", "-t", "One", "-c", "one")
	runCLI(t, sp, "", "add", "-t", "Two", "-c", "two")

	code, out, _ := runCLI(t, sp, "", "list", "--json")
	if code != 0 {
		t.Fatalf("list --json failed code=%d", code)
	}
	if !strings.Contains(out, "\"title\": \"One\"") || !strings.Contains(out, "\"title\": \"Two\"") {
		t.Fatalf("json missing snippets: %q", out)
	}
}

func TestShowMissing(t *testing.T) {
	sp := newStorePath(t)
	code, _, errb := runCLI(t, sp, "", "show", "doesnotexist")
	if code == 0 {
		t.Fatal("expected nonzero exit for missing snippet")
	}
	if !strings.Contains(errb, "no snippet") {
		t.Fatalf("unexpected error: %q", errb)
	}
}

func TestDeleteByID(t *testing.T) {
	sp := newStorePath(t)
	_, out, _ := runCLI(t, sp, "", "add", "-t", "Doomed", "-c", "x")
	id := strings.Fields(out)[1] // "added <id>  <title>"

	code, dout, _ := runCLI(t, sp, "", "delete", id)
	if code != 0 {
		t.Fatalf("delete failed code=%d", code)
	}
	if !strings.Contains(dout, "deleted") {
		t.Fatalf("unexpected delete output: %q", dout)
	}

	_, lout, _ := runCLI(t, sp, "", "list")
	if strings.Contains(lout, id) {
		t.Fatalf("snippet still present after delete: %q", lout)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	sp := newStorePath(t)
	runCLI(t, sp, "", "add", "-t", "Alpha", "-l", "go", "-c", "a")
	runCLI(t, sp, "", "add", "-t", "Beta", "-l", "bash", "-c", "b")

	exportFile := filepath.Join(t.TempDir(), "backup.json")
	code, _, errb := runCLI(t, sp, "", "export", "-o", exportFile)
	if code != 0 {
		t.Fatalf("export failed: %s", errb)
	}
	if _, err := os.Stat(exportFile); err != nil {
		t.Fatalf("export file not written: %v", err)
	}

	sp2 := newStorePath(t)
	code, out, _ := runCLI(t, sp2, "", "import", exportFile)
	if code != 0 {
		t.Fatalf("import failed code=%d", code)
	}
	if !strings.Contains(out, "imported 2") {
		t.Fatalf("unexpected import output: %q", out)
	}
	_, lout, _ := runCLI(t, sp2, "", "list")
	if !strings.Contains(lout, "Alpha") || !strings.Contains(lout, "Beta") {
		t.Fatalf("imported store missing snippets: %q", lout)
	}
}

func TestImportBareArray(t *testing.T) {
	sp := newStorePath(t)
	bare := `[{"title":"Bare","content":"x","language":"go"}]`
	f := filepath.Join(t.TempDir(), "bare.json")
	if err := os.WriteFile(f, []byte(bare), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runCLI(t, sp, "", "import", f)
	if code != 0 {
		t.Fatalf("import bare array failed: %s", errb)
	}
	if !strings.Contains(out, "imported 1") {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestVersion(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"version"}, strings.NewReader(""), &out, &out)
	if code != 0 {
		t.Fatalf("version exit=%d", code)
	}
	if !strings.Contains(out.String(), "snippetbox") {
		t.Fatalf("unexpected version output: %q", out.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"frobnicate"}, strings.NewReader(""), &out, &errb)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(errb.String(), "unknown command") {
		t.Fatalf("unexpected stderr: %q", errb.String())
	}
}

func TestExtractStoreFlagVariants(t *testing.T) {
	cases := []struct {
		args  []string
		store string
		rest  []string
	}{
		{[]string{"--store", "p", "list"}, "p", []string{"list"}},
		{[]string{"--store=p", "list"}, "p", []string{"list"}},
		{[]string{"list", "--store", "p"}, "p", []string{"list"}},
		{[]string{"list"}, "", []string{"list"}},
	}
	for _, c := range cases {
		gotStore, gotRest := extractStoreFlag(c.args)
		if gotStore != c.store {
			t.Errorf("args %v: store=%q want %q", c.args, gotStore, c.store)
		}
		if strings.Join(gotRest, ",") != strings.Join(c.rest, ",") {
			t.Errorf("args %v: rest=%v want %v", c.args, gotRest, c.rest)
		}
	}
}

func TestShowSanitizesEscape(t *testing.T) {
	sp := newStorePath(t)
	// Add a snippet whose content carries terminal escape sequences via stdin so
	// nothing rewrites it on the way in.
	payload := "echo hi\x1b[2J\x1b]0;PWNED\x07done"
	code, _, errb := runCLI(t, sp, payload, "add", "-t", "Evil", "--stdin")
	if code != 0 {
		t.Fatalf("add failed: %s", errb)
	}
	code, out, _ := runCLI(t, sp, "", "show", "Evil")
	if code != 0 {
		t.Fatalf("show failed code=%d", code)
	}
	if strings.ContainsRune(out, 0x1b) {
		t.Fatalf("show emitted raw ESC byte: %q", out)
	}
	if strings.ContainsRune(out, 0x07) {
		t.Fatalf("show emitted raw BEL byte: %q", out)
	}
	if !strings.Contains(out, "echo hi") || !strings.Contains(out, "done") {
		t.Fatalf("show dropped visible content: %q", out)
	}
}

func TestListSanitizesEscape(t *testing.T) {
	sp := newStorePath(t)
	runCLI(t, sp, "ls\x1b]0;PWN\x07", "add", "-t", "ti\x1b[2Jtle", "--stdin")
	_, out, _ := runCLI(t, sp, "", "list")
	if strings.ContainsRune(out, 0x1b) || strings.ContainsRune(out, 0x07) {
		t.Fatalf("list emitted raw control bytes: %q", out)
	}
}

func TestImportSanitizesContent(t *testing.T) {
	sp := newStorePath(t)
	bad := `[{"title":"X\u001b[2J","content":"echo\u001b]0;PWN\u0007 hi"}]`
	f := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(f, []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runCLI(t, sp, "", "import", f)
	if code != 0 {
		t.Fatalf("import failed: %s", errb)
	}
	if !strings.Contains(out, "imported 1") {
		t.Fatalf("unexpected import output: %q", out)
	}
	// Read the store file directly: no ESC/BEL must have been persisted.
	data, err := os.ReadFile(sp)
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	if strings.ContainsRune(string(data), 0x1b) || strings.ContainsRune(string(data), 0x07) {
		t.Fatalf("control byte persisted to store: %q", string(data))
	}
}

func TestImportRejectsInvalidAndGarbage(t *testing.T) {
	sp := newStorePath(t)

	// Snippets missing title/content are skipped, not imported.
	skip := `[{"title":"","content":"x"},{"title":"y","content":""}]`
	f := filepath.Join(t.TempDir(), "skip.json")
	if err := os.WriteFile(f, []byte(skip), 0o600); err != nil {
		t.Fatal(err)
	}
	_, out, _ := runCLI(t, sp, "", "import", f)
	if !strings.Contains(out, "imported 0") {
		t.Fatalf("expected 0 imported for invalid snippets, got %q", out)
	}

	// Non-array, non-document JSON is rejected outright.
	garbage := `{"unexpected":"object"}`
	g := filepath.Join(t.TempDir(), "garbage.json")
	if err := os.WriteFile(g, []byte(garbage), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, errb := runCLI(t, sp, "", "import", g)
	if code == 0 {
		t.Fatalf("expected nonzero exit for garbage import, stderr=%q", errb)
	}
	if !strings.Contains(errb, "parse import file") {
		t.Fatalf("unexpected error: %q", errb)
	}
}

func TestImportRejectsOversize(t *testing.T) {
	sp := newStorePath(t)
	big := filepath.Join(t.TempDir(), "big.json")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	// Write just over the 16 MiB limit.
	if _, err := f.Write([]byte("[")); err != nil {
		t.Fatal(err)
	}
	chunk := make([]byte, 1<<20)
	for i := range chunk {
		chunk[i] = ' '
	}
	for i := 0; i < 17; i++ {
		if _, err := f.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()

	code, _, errb := runCLI(t, sp, "", "import", big)
	if code == 0 {
		t.Fatal("expected nonzero exit for oversize import")
	}
	if !strings.Contains(errb, "limit") {
		t.Fatalf("expected size-limit error, got %q", errb)
	}
}

func TestExportPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file mode bits are not meaningful on Windows")
	}
	sp := newStorePath(t)
	runCLI(t, sp, "", "add", "-t", "S", "-c", "secret")
	out := filepath.Join(t.TempDir(), "export.json")
	if code, _, errb := runCLI(t, sp, "", "export", "-o", out); code != 0 {
		t.Fatalf("export failed: %s", errb)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat export: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("export mode = %o, want 0600", got)
	}
}

func TestFindLimit(t *testing.T) {
	sp := newStorePath(t)
	for _, name := range []string{"alpha one", "alpha two", "alpha three"} {
		runCLI(t, sp, "", "add", "-t", name, "-c", "x")
	}
	_, out, _ := runCLI(t, sp, "", "find", "-n", "2", "alpha")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines with -n 2, got %d: %q", len(lines), out)
	}
}
