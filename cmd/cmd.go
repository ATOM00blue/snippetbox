// Package cmd implements snippetbox's command-line interface. With no
// subcommand it launches the TUI; otherwise it dispatches to scriptable
// subcommands (add, find, copy, show, list, delete, import, export, version).
package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ATOM00blue/snippetbox/internal/sanitize"
	"github.com/ATOM00blue/snippetbox/internal/search"
	"github.com/ATOM00blue/snippetbox/internal/snippet"
	"github.com/ATOM00blue/snippetbox/internal/store"
	"github.com/ATOM00blue/snippetbox/internal/tui"
	"github.com/atotto/clipboard"
)

// maxImportBytes caps the size of an import file to bound memory use from a
// hostile or accidentally huge file.
const maxImportBytes = 16 << 20 // 16 MiB

// Version is the build version, overridable at link time:
//
//	go build -ldflags "-X github.com/ATOM00blue/snippetbox/cmd.Version=v1.2.3"
var Version = "dev"

// Execute runs the CLI and returns a process exit code.
func Execute() int {
	return run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}

// run is the testable core of Execute. It never calls os.Exit.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// Extract a global --store flag before subcommand parsing so it can appear
	// anywhere on the command line.
	storePath, args := extractStoreFlag(args)

	if len(args) == 0 {
		return runTUI(storePath, stderr)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "add":
		return cmdAdd(storePath, rest, stdin, stdout, stderr)
	case "find", "search":
		return cmdFind(storePath, rest, stdout, stderr)
	case "copy", "cp":
		return cmdCopy(storePath, rest, stdout, stderr)
	case "show", "cat":
		return cmdShow(storePath, rest, stdout, stderr)
	case "list", "ls":
		return cmdList(storePath, rest, stdout, stderr)
	case "delete", "rm":
		return cmdDelete(storePath, rest, stdout, stderr)
	case "export":
		return cmdExport(storePath, rest, stdout, stderr)
	case "import":
		return cmdImport(storePath, rest, stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, "snippetbox", Version)
		return 0
	case "help", "--help", "-h":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "snippetbox: unknown command %q\n\n", sub)
		usage(stderr)
		return 2
	}
}

// extractStoreFlag pulls "--store PATH" / "--store=PATH" (and -store variants)
// out of args, returning the path and the remaining args.
func extractStoreFlag(args []string) (string, []string) {
	out := make([]string, 0, len(args))
	storePath := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--store" || a == "-store":
			if i+1 < len(args) {
				storePath = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--store="):
			storePath = strings.TrimPrefix(a, "--store=")
		case strings.HasPrefix(a, "-store="):
			storePath = strings.TrimPrefix(a, "-store=")
		default:
			out = append(out, a)
		}
	}
	return storePath, out
}

func openStore(path string, stderr io.Writer) (*store.Store, bool) {
	s, err := store.Open(path)
	if err != nil {
		fmt.Fprintln(stderr, "snippetbox:", err)
		return nil, false
	}
	return s, true
}

func runTUI(storePath string, stderr io.Writer) int {
	s, ok := openStore(storePath, stderr)
	if !ok {
		return 1
	}
	if err := tui.Run(s); err != nil {
		fmt.Fprintln(stderr, "snippetbox:", err)
		return 1
	}
	return 0
}

func cmdAdd(storePath string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	title := fs.String("t", "", "snippet title (required)")
	lang := fs.String("l", "", "language (e.g. bash, go, python)")
	tags := fs.String("T", "", "comma-separated tags")
	content := fs.String("c", "", "snippet content")
	fromStdin := fs.Bool("stdin", false, "read content from stdin")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*title) == "" {
		fmt.Fprintln(stderr, "snippetbox: add requires a title (-t)")
		return 2
	}
	body := *content
	if *fromStdin {
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintln(stderr, "snippetbox: read stdin:", err)
			return 1
		}
		body = string(data)
	}
	if strings.TrimSpace(body) == "" {
		fmt.Fprintln(stderr, "snippetbox: add requires content (-c or --stdin)")
		return 2
	}

	s, ok := openStore(storePath, stderr)
	if !ok {
		return 1
	}
	sn := snippet.New(*title, *lang, body, snippet.ParseTags(*tags))
	saved, err := s.Add(sn)
	if err != nil {
		fmt.Fprintln(stderr, "snippetbox:", err)
		return 1
	}
	fmt.Fprintf(stdout, "added %s  %s\n", saved.ID, saved.Title)
	return 0
}

func cmdFind(storePath string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output results as JSON")
	limit := fs.Int("n", 0, "limit number of results (0 = all)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	query := strings.Join(fs.Args(), " ")

	s, ok := openStore(storePath, stderr)
	if !ok {
		return 1
	}
	results := search.Search(s.All(), query)
	if *limit > 0 && len(results) > *limit {
		results = results[:*limit]
	}
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(search.Snippets(results)); err != nil {
			fmt.Fprintln(stderr, "snippetbox:", err)
			return 1
		}
		return 0
	}
	for _, r := range results {
		fmt.Fprintf(stdout, "%s\t%s\n", r.Snippet.ID, sanitize.Line(r.Snippet.Summary()))
	}
	return 0
}

func cmdCopy(storePath string, args []string, stdout, stderr io.Writer) int {
	query := strings.Join(args, " ")
	if strings.TrimSpace(query) == "" {
		fmt.Fprintln(stderr, "snippetbox: copy requires a query or id")
		return 2
	}
	s, ok := openStore(storePath, stderr)
	if !ok {
		return 1
	}
	sn, found := resolve(s, query)
	if !found {
		fmt.Fprintf(stderr, "snippetbox: no snippet matching %q\n", query)
		return 1
	}
	if err := clipboard.WriteAll(sn.Content); err != nil {
		fmt.Fprintln(stderr, "snippetbox: copy to clipboard failed:", err)
		return 1
	}
	fmt.Fprintf(stdout, "copied %s  %s\n", sn.ID, sn.Title)
	return 0
}

func cmdShow(storePath string, args []string, stdout, stderr io.Writer) int {
	query := strings.Join(args, " ")
	if strings.TrimSpace(query) == "" {
		fmt.Fprintln(stderr, "snippetbox: show requires a query or id")
		return 2
	}
	s, ok := openStore(storePath, stderr)
	if !ok {
		return 1
	}
	sn, found := resolve(s, query)
	if !found {
		fmt.Fprintf(stderr, "snippetbox: no snippet matching %q\n", query)
		return 1
	}
	// Strip terminal control sequences before printing untrusted content.
	fmt.Fprintln(stdout, sanitize.Content(sn.Content))
	return 0
}

func cmdList(storePath string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	s, ok := openStore(storePath, stderr)
	if !ok {
		return 1
	}
	all := s.All()
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(all); err != nil {
			fmt.Fprintln(stderr, "snippetbox:", err)
			return 1
		}
		return 0
	}
	for _, sn := range all {
		fmt.Fprintf(stdout, "%s\t%s\n", sn.ID, sanitize.Line(sn.Summary()))
	}
	return 0
}

func cmdDelete(storePath string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "snippetbox: delete requires a snippet id")
		return 2
	}
	s, ok := openStore(storePath, stderr)
	if !ok {
		return 1
	}
	code := 0
	for _, id := range args {
		if err := s.Delete(id); err != nil {
			fmt.Fprintf(stderr, "snippetbox: %s: %v\n", id, err)
			code = 1
			continue
		}
		fmt.Fprintln(stdout, "deleted", id)
	}
	return code
}

func cmdExport(storePath string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("o", "", "write to file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	s, ok := openStore(storePath, stderr)
	if !ok {
		return 1
	}
	data, err := s.Marshal()
	if err != nil {
		fmt.Fprintln(stderr, "snippetbox:", err)
		return 1
	}
	if *out == "" {
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	// The export is a full copy of the store and may contain secrets, so it is
	// written owner read/write only.
	if err := os.WriteFile(*out, data, 0o600); err != nil {
		fmt.Fprintln(stderr, "snippetbox:", err)
		return 1
	}
	fmt.Fprintf(stdout, "exported %d snippet(s) to %s\n", s.Len(), *out)
	return 0
}

func cmdImport(storePath string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "snippetbox: import requires a file path")
		return 2
	}
	data, err := readImportFile(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "snippetbox:", err)
		return 1
	}
	incoming, err := decodeSnippets(data)
	if err != nil {
		fmt.Fprintln(stderr, "snippetbox: parse import file:", err)
		return 1
	}
	s, ok := openStore(storePath, stderr)
	if !ok {
		return 1
	}
	added, err := s.Import(incoming)
	if err != nil {
		fmt.Fprintln(stderr, "snippetbox:", err)
		return 1
	}
	fmt.Fprintf(stdout, "imported %d snippet(s)\n", added)
	return 0
}

// readImportFile reads an import file, refusing files larger than maxImportBytes
// to bound memory use from a hostile or accidentally huge file.
func readImportFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Read one byte past the cap so we can detect oversize files.
	data, err := io.ReadAll(io.LimitReader(f, maxImportBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxImportBytes {
		return nil, fmt.Errorf("import file exceeds %d MiB limit", maxImportBytes>>20)
	}
	return data, nil
}

// decodeSnippets accepts either a full store document {"version":..,"snippets":[]}
// or a bare array of snippets. Any other JSON shape (object, scalar, garbage) is
// rejected so a malformed file cannot be silently treated as zero snippets.
func decodeSnippets(data []byte) ([]snippet.Snippet, error) {
	var doc struct {
		Snippets []snippet.Snippet `json:"snippets"`
	}
	if err := json.Unmarshal(data, &doc); err == nil && doc.Snippets != nil {
		return doc.Snippets, nil
	}
	var arr []snippet.Snippet
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("expected a snippet array or store document: %w", err)
	}
	return arr, nil
}

// resolve finds a snippet by exact ID first, then falls back to best fuzzy match.
func resolve(s *store.Store, query string) (snippet.Snippet, bool) {
	if sn, err := s.Get(query); err == nil {
		return sn, true
	}
	return search.Best(s.All(), query)
}

func usage(w io.Writer) {
	const help = `snippetbox - a fast terminal snippet manager

USAGE:
  snippetbox                 launch the interactive TUI
  snippetbox <command> [...]

COMMANDS:
  add      add a snippet         -t TITLE [-l LANG] [-T tag,tag] (-c BODY | --stdin)
  find     fuzzy search          find [QUERY] [-n N] [--json]
  copy     copy match to clipboard   copy QUERY|ID
  show     print snippet content     show QUERY|ID
  list     list all snippets         list [--json]
  delete   delete by id              delete ID [ID...]
  export   dump store JSON           export [-o FILE]
  import   merge snippets from file  import FILE
  version  print version
  help     show this help

GLOBAL FLAGS:
  --store PATH    use an alternate store file (env: SNIPPETBOX_STORE)

EXAMPLES:
  snippetbox add -t "Tar dir" -l bash -T archive -c 'tar -czf out.tgz ./dir'
  cat main.go | snippetbox add -t "main" -l go --stdin
  snippetbox find tar
  snippetbox copy tar
`
	fmt.Fprint(w, help)
}
