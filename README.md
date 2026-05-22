<div align="center">

# snippetbox

**A fast terminal snippet manager with fuzzy search, syntax-highlighted preview, and one-key copy to clipboard.**

[![CI](https://github.com/ATOM00blue/snippetbox/actions/workflows/ci.yml/badge.svg)](https://github.com/ATOM00blue/snippetbox/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ATOM00blue/snippetbox.svg)](https://pkg.go.dev/github.com/ATOM00blue/snippetbox)
[![Go Report Card](https://goreportcard.com/badge/github.com/ATOM00blue/snippetbox)](https://goreportcard.com/report/github.com/ATOM00blue/snippetbox)
[![Release](https://img.shields.io/github/v/release/ATOM00blue/snippetbox?include_prereleases&sort=semver)](https://github.com/ATOM00blue/snippetbox/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Stop digging through old projects and chat logs for that one command. Keep your
snippets one keystroke away, right in the terminal.

</div>

---

## Demo

```text
┌ snippetbox ──────────────────┐ ╭ Tar a directory ──────────────────────────╮
│ > Tar a directory            │ │ id 6535eq • lang bash • #archive #shell    │
│   bash  #archive #shell      │ │                                            │
│                              │ │ tar -czf out.tgz ./dir                     │
│   Go HTTP server             │ │                                            │
│   go  #web                   │ │                                            │
│                              │ │                                            │
│   List files                 │ │                                            │
│   bash                       │ │                                            │
└──────────────────────────────┘ ╰────────────────────────────────────────────╯
 / filter • a add • e edit • d delete • c copy • tab focus • q quit
```

> Replace this ASCII mock with a real screenshot/GIF: `docs/demo.gif`.

## Features

- **Fuzzy search across everything** — title, tags, language *and* content, not
  just titles. Find that snippet even when you only remember a word from inside it.
- **Syntax-highlighted preview** — powered by [chroma](https://github.com/alecthomas/chroma);
  language auto-detected when not set.
- **One-key copy to clipboard** — press `c` in the TUI (or `snippetbox copy <query>`
  from a script). Cross-platform via [atotto/clipboard](https://github.com/atotto/clipboard).
- **Full CRUD in the TUI** — add, edit, delete without leaving the app.
- **First-class scripting** — every action has a non-interactive subcommand with
  clean, pipeable output (`add`, `find`, `copy`, `show`, `list`, `delete`,
  `import`, `export`).
- **Git-friendly storage** — a single JSON file in your config dir. Easy to sync,
  diff, and back up. Writes are atomic, so a crash never corrupts your snippets.
- **Single static binary** — `go install` and you're done. No runtime deps.

## Install

### `go install` (recommended)

```sh
go install github.com/ATOM00blue/snippetbox@latest
```

This drops a `snippetbox` binary in `$(go env GOPATH)/bin` (make sure it's on
your `PATH`).

### Pre-built binaries

Grab a binary for your OS from the
[Releases page](https://github.com/ATOM00blue/snippetbox/releases) and put it on
your `PATH`.

### From source

```sh
git clone https://github.com/ATOM00blue/snippetbox.git
cd snippetbox
go build -o snippetbox .
```

## Usage

Launch the interactive TUI:

```sh
snippetbox
```

### Keybindings

| Key            | Action                                   |
| -------------- | ---------------------------------------- |
| `↑` / `k`      | move up                                  |
| `↓` / `j`      | move down                                |
| `/`            | filter (fuzzy across title/tags/content) |
| `enter`        | apply filter                             |
| `esc`          | clear filter / cancel                    |
| `tab`          | toggle focus (list ⇄ preview scroll)     |
| `c`            | copy selected snippet to clipboard       |
| `a`            | add a new snippet                        |
| `e`            | edit selected snippet                    |
| `d` then `y`   | delete selected snippet (with confirm)   |
| `q` / `ctrl+c` | quit                                     |

In the add/edit form: `tab` / `↑` `↓` move between fields, `ctrl+s` saves, `esc`
cancels.

### Scripting subcommands

snippetbox is built to be used from scripts and your shell, not just the TUI:

```sh
# Add a snippet with flags
snippetbox add -t "Tar a directory" -l bash -T archive,shell -c 'tar -czf out.tgz ./dir'

# Pipe content from stdin (great for whole files)
cat main.go | snippetbox add -t "Go main" -l go --stdin

# Fuzzy search (TSV: <id>\t<summary>)
snippetbox find tar
snippetbox find -n 5 --json server   # top 5 matches as JSON

# Print the raw content of the best match (pipe it anywhere)
snippetbox show tar | bash

# Copy the best match straight to your clipboard
snippetbox copy tar

# Manage
snippetbox list
snippetbox delete 6535eq
snippetbox export -o backup.json
snippetbox import backup.json
```

Both `copy` and `show` accept either an exact snippet **id** or a **fuzzy
query** (id is tried first, then the best fuzzy match).

### Storage location

Snippets live in a single JSON file in your OS config directory:

| OS      | Default path                                       |
| ------- | -------------------------------------------------- |
| Linux   | `$XDG_CONFIG_HOME/snippetbox/snippets.json` (or `~/.config/...`) |
| macOS   | `~/Library/Application Support/snippetbox/snippets.json` |
| Windows | `%AppData%\snippetbox\snippets.json`               |

Override the path with the global `--store <path>` flag or the
`SNIPPETBOX_STORE` environment variable — handy for project-local snippet sets
or for scripting/testing:

```sh
SNIPPETBOX_STORE=./project-snippets.json snippetbox add -t "Deploy" -c "make deploy"
```

### Security notes

- **Permissions.** Because snippets often contain secrets, the store file is
  written `0600` (owner read/write only) and its directory `0700`, using an
  atomic temp-file + rename. `export -o FILE` is likewise written `0600`. (On
  Windows, Go maps these onto the platform's ACL model.)
- **Terminal-safe rendering.** Snippet content can come from anywhere (imports,
  pasted text). snippetbox strips terminal escape/control sequences from any
  content it renders to your terminal — in the TUI preview and list, and in the
  `show` / `list` / `find` output — so a hostile snippet can't hijack your
  terminal. Legitimate newlines and tabs are preserved. The `--json` outputs and
  `export` emit data, not terminal output, and are JSON-escaped.
- **Import limits.** `import` rejects files over 16 MiB, validates every snippet,
  and skips entries with no title or content.
- **No code execution.** snippetbox never executes snippet content. `show` simply
  prints it; piping it into a shell (`snippetbox show foo | bash`) is your choice.

## FAQ

**Is this the "Let's Go" / Alex Edwards tutorial web app?**
No. That's an unrelated Go *web* tutorial that happens to share the name. This is
a terminal snippet manager.

**How is it different from `pet` or `nap`?**
`pet` focuses on shell commands with parameter expansion; `nap` is a great TUI
but filters mainly on titles/folders and shells out to `$EDITOR`. snippetbox
fuzzy-searches across *content* too, highlights the preview, edits inline, and
ships strong scriptable subcommands so it slots into pipelines.

**Where does the clipboard support come from?**
[`atotto/clipboard`](https://github.com/atotto/clipboard). On Linux you may need
`xsel`, `xclip`, `wl-clipboard`, or `termux-clipboard-set` installed.

**Can I sync my snippets?**
Yes — point `SNIPPETBOX_STORE` at a file in a synced/Git-tracked folder, or just
commit the JSON file. It's plain JSON with atomic writes, so it diffs cleanly.

**Does it send anything over the network?**
No. Everything is local.

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md). In short:

```sh
go test ./...   # all green
go vet ./...    # clean
gofmt -l .      # no output
```

## License

[MIT](LICENSE) © 2026 ATOM00blue

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Lip Gloss](https://github.com/charmbracelet/lipgloss),
[chroma](https://github.com/alecthomas/chroma), and
[sahilm/fuzzy](https://github.com/sahilm/fuzzy).
