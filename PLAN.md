# snippetbox — PLAN

A fast terminal snippet manager with fuzzy search, syntax-highlighted preview,
and one-key copy to clipboard. Single static Go binary.

## Research summary

Competitors:
- **pet** (knqyf263/pet) — CLI-first, focused on *shell command* snippets with
  parameter expansion. No rich TUI, no syntax highlighting.
- **nap** (maaslalani/nap) — TUI + CLI for code snippets. Folders + languages.
  Strong UX but uses an external `$EDITOR` for all editing and has no built-in
  full-text fuzzy search across snippet *content* (filters titles/folders).
- **snibox** — self-hosted web app (Vue + Ruby). Not a terminal tool.
- **massren** — bulk rename tool; only loosely related (editor-driven editing).

Gaps we exploit:
1. **Fuzzy search across title + tags + content + language** (most tools only
   match titles). Powered by `sahilm/fuzzy`.
2. **Syntax-highlighted preview pane** in the TUI using `chroma`.
3. **First-class non-interactive subcommands** (`add`, `find`, `copy`, `list`,
   `delete`, `show`, `import`, `export`) so it scripts cleanly — pet/nap are
   weaker here for content search.
4. **One-key copy** to the system clipboard from the TUI (`atotto/clipboard`).
5. **Single JSON store** in the OS config dir, trivially syncable / git-friendly.

## Tech stack & libraries

- Go 1.21+
- TUI: `charmbracelet/bubbletea` v1.3.10, `bubbles` v0.21.0, `lipgloss` v1.1.0
- Fuzzy: `sahilm/fuzzy` v0.1.1
- Highlight: `alecthomas/chroma/v2` v2.24.1
- Clipboard: `atotto/clipboard` v0.1.4

Module path: `github.com/ATOM00blue/snippetbox`

## Storage format

JSON file at the OS config dir:
- Linux:   `$XDG_CONFIG_HOME/snippetbox/snippets.json` (or `~/.config/...`)
- macOS:   `~/Library/Application Support/snippetbox/snippets.json`
- Windows: `%AppData%\snippetbox\snippets.json`

Override with `--store <path>` flag or `SNIPPETBOX_STORE` env var (used by tests
and scripting for isolation).

```json
{
  "version": 1,
  "snippets": [
    {
      "id": "k4f9a2",
      "title": "Tar a directory",
      "tags": ["bash", "archive"],
      "language": "bash",
      "content": "tar -czf out.tgz ./dir",
      "created_at": "2026-05-22T10:00:00Z",
      "updated_at": "2026-05-22T10:00:00Z"
    }
  ]
}
```

Writes are atomic (temp file + rename). IDs are short random base36 strings.

## File layout

```
snippetbox/
├── main.go                  # thin entry point -> cmd.Execute
├── internal/
│   ├── snippet/
│   │   └── snippet.go       # Snippet model + helpers
│   ├── store/
│   │   ├── store.go         # load/save/CRUD, atomic write, default path
│   │   └── store_test.go
│   ├── search/
│   │   ├── search.go        # fuzzy search over snippets
│   │   └── search_test.go
│   └── tui/
│       ├── tui.go           # Bubble Tea model: list + preview + filter
│       ├── form.go          # add/edit form
│       └── highlight.go     # chroma highlighting helpers
├── cmd/
│   ├── cmd.go               # arg parsing / dispatch (stdlib flag based)
│   └── cmd_test.go
├── PLAN.md / README.md / LICENSE / CONTRIBUTING.md / CHANGELOG.md
├── .gitignore
└── .github/workflows/ci.yml
```

## CLI surface

```
snippetbox                          launch the TUI
snippetbox add -t TITLE [-l LANG] [-T tag,tag] [-c CONTENT | --stdin]
snippetbox find [QUERY]             fuzzy search, print matches (TSV-ish)
snippetbox copy QUERY               copy best match's content to clipboard
snippetbox show QUERY|ID            print a snippet's content (raw)
snippetbox list                     list all snippets
snippetbox delete ID                delete by id
snippetbox export [-o file.json]    dump store JSON to stdout/file
snippetbox import file.json         merge snippets from a file
snippetbox version                  print version
Flags: --store PATH (global), env SNIPPETBOX_STORE
```

## TUI screen / state design

Single screen, two panes:
- Left: filterable list (bubbles/list) of snippets (title + lang + tags).
- Right: viewport preview with chroma syntax highlighting.
- Bottom: help/keybinding bar.

Modes: `browsing`, `filtering` (handled by list), `form` (add/edit overlay),
`confirm-delete`.

Keybindings:
- `/` filter, `enter` apply filter, `esc` clear
- `↑/↓` or `j/k` navigate; `tab` focus toggle (list/preview scroll)
- `c` copy current snippet to clipboard
- `a` add, `e` edit, `d` delete (with confirm `y/n`)
- `?` toggle full help, `q`/`ctrl+c` quit

## Standout features (beyond MVP)

- Content-level fuzzy search (not just titles).
- Syntax-highlighted preview (auto-detect language via chroma if unset).
- Scriptable subcommands incl. `copy`/`show`/`find` for pipelines.
- Atomic, git-friendly single-file JSON store with import/export.
- Cross-platform clipboard + config dir handling, single static binary.

## Verification plan

- `go vet ./...` clean.
- `go test ./...` green (store CRUD + atomic write + search ranking + cmd).
- Smoke: build binary, `add` then `find`/`show` against a temp store inside the
  repo, assert output.
- CI on Linux/macOS/Windows: build, vet, test.
