package cmd

import (
	"bytes"
	"os"
	"path/filepath"
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
