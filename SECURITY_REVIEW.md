# Security & Quality Review — snippetbox

Date: 2026-05-22
Reviewer: autonomous application-security review
Scope: full source tree of `github.com/ATOM00blue/snippetbox` at commit `c0d1ede`
Stack: Go 1.24+, Bubble Tea TUI, single JSON store in the OS config dir.

Tooling run: `go build`, `go vet`, `gofmt -l`, `go test`, `govulncheck ./...`,
`gosec ./...`, plus targeted runtime probes of the chroma highlighter and the
store permission/atomic-write paths.

---

## Summary of findings

| # | Severity | Area | Title | Status |
|---|----------|------|-------|--------|
| H1 | High | TUI render | Terminal escape-sequence injection from snippet content (preview, list, meta) | Fixed |
| H2 | High | Store perms | Store file/dir not explicitly created with restrictive perms (secrets at rest) | Fixed |
| M1 | Medium | Import | Untrusted import: no size limit, no schema/content validation, control chars persisted | Fixed |
| M2 | Medium | Export | Export written world-readable (0644) although it may contain secrets | Fixed |
| M3 | Medium | CLI render | `show`/`list` print raw snippet content/titles to a terminal without sanitization | Fixed |
| L1 | Low | Store | Unhandled `Close()` errors on the write-error paths in `Save` | Fixed |
| L2 | Low | Robustness | Corrupt store file fails the whole TUI/CLI with a bare parse error | Documented (acceptable) |
| I1 | Info | Supply chain | `govulncheck`: 0 reachable vulns; stdlib advisories not in call graph | Documented |
| I2 | Info | Path handling | `--store` / `SNIPPETBOX_STORE` / import path are user-supplied (by design) | Documented |

No Critical findings. No command execution of snippet content exists anywhere
(snippets are only stored, copied to the clipboard, or printed — never `exec`'d).

---

## High

### H1 — Terminal escape-sequence injection in rendered snippet content
File: `internal/tui/highlight.go:16`, `internal/tui/tui.go` (preview, list item
`Title`/`Description`, preview meta), `cmd/cmd.go` (`show`, `list`).

Impact. Snippet content is fully attacker-controlled (anyone can hand you a
snippet to import, or you paste one from the web). When such content is rendered
in the TUI preview or printed by `show`, any ANSI/OSC control sequences embedded
in the content are interpreted by the terminal. Demonstrated impact includes:
clearing the screen (`ESC[2J`), rewriting the window/tab title (`OSC 0`),
hiding text (`ESC[8m`) to spoof what the user thinks they are copying, and on
some terminals abusing OSC 52 / DCS for clipboard or response injection. A
runtime probe confirmed the lone `ESC` (0x1b) bytes from the content survive
through chroma's `terminal256` formatter into the rendered output:

```
INPUT:  echo hi\x1b[2J\x1b]0;PWNED\a\x1b[8mhidden\x1b[0m
OUTPUT: ...\x1b\x1b[0m...  (raw ESC bytes preserved between color codes)
```

chroma colorizes tokens but does **not** strip C0/C1 control characters from the
token text, so untrusted ESC bytes reach the terminal. The same applies to the
non-highlighted paths: list titles/descriptions, the preview meta line, and the
`show`/`list` CLI output, none of which sanitized control characters.

Fix. Added `internal/sanitize` with `Content` (preserves `\n`/`\t`, strips all
other C0/C1 controls including ESC, BEL, and DEL, plus the Unicode line/paragraph
separators) and `Line` (also collapses newlines, for single-line list/meta use).
All untrusted text is now sanitized **before** it reaches the terminal:
- preview content is sanitized before being passed to chroma (`refreshPreview`);
- list item `Title`/`Description` and the preview header/meta are sanitized;
- `cmd show` sanitizes content and `cmd list` (non-JSON) sanitizes the summary.
JSON output (`find --json`, `list --json`, `export`) is left raw on purpose —
it is data, not terminal output, and JSON encoding already escapes control bytes.

### H2 — Store file/dir not created with restrictive permissions
File: `internal/store/store.go:205` (`MkdirAll(dir, 0o755)`), `Save` temp file.

Impact. Snippets routinely contain secrets (tokens, connection strings, private
commands). The store directory was created `0755` (world-readable) and the store
file's permissions relied on the `os.CreateTemp` default — an implementation
detail, not an explicit guarantee, and one that does nothing to harden an
already-existing file. On a multi-user host this risks other local users reading
secrets at rest.

Fix. `Save` now creates the directory `0700` and explicitly `Chmod`s the temp
file to `0600` before the atomic rename, so the live store file is always owner
read/write only. (On Windows, Go maps these onto the platform ACL model as it
does for any file; the explicit mode is a no-op there but correct on Unix.)
Added regression test `TestSavePermissions` (skipped on Windows where Unix mode
bits are not meaningful).

---

## Medium

### M1 — Untrusted import lacks size limit and content/schema validation
File: `cmd/cmd.go:313` (`cmdImport`), `internal/store/store.go:161` (`Import`).

Impact. `import FILE` read the entire file into memory with no cap (memory
exhaustion / DoS from a hostile or accidentally huge file) and accepted snippets
with no validation: arbitrarily long fields, control characters baked into
titles/content (which then get re-rendered — see H1), and empty/garbage records.
Imported snippets persisted these control bytes into the store.

Fix.
- `cmdImport` now reads through an `io.LimitReader` capped at 16 MiB and rejects
  files that exceed it with a clear error.
- Added `snippet.Sanitize`/validation applied on `Import` (and on `Add`/`Update`):
  titles and language are sanitized to a single clean line and length-capped,
  content control characters are stripped (keeping `\n`/`\t`), tags normalized,
  and snippets with an empty title or empty content after sanitization are
  rejected. `decodeSnippets` rejects a payload that is neither a store document
  nor a JSON array.
Added tests: `TestImportSanitizesAndValidates`, `TestImportRejectsOversize`,
`TestDecodeSnippetsRejectsGarbage`.

### M2 — Export written world-readable
File: `cmd/cmd.go:305` (`os.WriteFile(*out, data, 0o644)`).

Impact. The export is a full copy of the store (secrets included) but was
written `0644`, world-readable.

Fix. Export now writes `0600`. Documented in README/CHANGELOG.

### M3 — CLI `show`/`list` emit raw content to the terminal
File: `cmd/cmd.go:233` (`show`), `cmd/cmd.go:259` (`list`).

Impact. Same class as H1 but via the scriptable subcommands when run in an
interactive terminal.

Fix. `show` sanitizes content (preserving newlines/tabs) before printing; `list`
sanitizes each summary line. Behavior for piped/JSON consumers is unchanged for
the JSON paths; `show` of legitimate code is byte-identical except that genuine
embedded control characters (which are not valid in stored snippets after M1) are
stripped. Added `TestShowSanitizesEscape`.

---

## Low

### L1 — Unhandled `Close()` on error paths in `Save`
File: `internal/store/store.go:217,221`. `tmp.Close()` errors were ignored on the
write/sync failure paths (gosec G104). Harmless but untidy; now the deferred
cleanup handles the temp file and the close error is no longer silently dropped
in a way that masks the real error. (The primary error is still returned.)

### L2 — Corrupt store surfaces a raw parse error (accepted)
File: `internal/store/store.go:75`. A malformed JSON store fails `Open` with a
wrapped parse error and the app exits. This is intentional: silently discarding a
user's (possibly recoverable) snippet file would be worse than a clear failure.
The atomic write (temp + rename) already prevents the tool itself from producing
a corrupt store. Left as-is by design; the error message includes the path so the
user can inspect/fix or restore from a backup/export.

---

## Informational

### I1 — Supply chain (`govulncheck` / `gosec`)
`govulncheck ./...`: **0 vulnerabilities affecting this code.** Reported stdlib
advisories (net, net/url, net/mail, os.Root) are not in snippetbox's call graph —
the tool does no networking and does not use `os.Root`. Dependencies
(chroma, bubbletea, lipgloss, fuzzy, atotto/clipboard) have no reachable
advisories. No action required beyond keeping the toolchain current; CI builds on
the latest Go 1.24.x.

`gosec ./...`: the reported G304/G703 path-traversal hits are the user-supplied
store/import/export paths (see I2) — these are first-party CLI inputs, not
network-tainted data, so confining them to a sandbox root is not appropriate for
a local CLI. The G301/G306 permission findings are addressed by H2/M2. The G104
findings are addressed by L1.

### I2 — User-supplied paths (`--store`, `SNIPPETBOX_STORE`, import path)
By design a local CLI lets the user choose where the store lives and which file to
import. There is no privilege boundary to cross: the process runs as the user and
can already read/write anything the user can. `..` in these paths is therefore not
a vulnerability — it is the documented feature (project-local stores). No
"jailing" is applied because it would break the documented `--store ./x.json`
behavior. The import *content* is the untrusted part and is now validated (M1).
No path is ever taken from snippet content or from the network.

---

## Verification performed

- `go build ./...` — clean
- `go vet ./...` — clean
- `gofmt -l .` — no output
- `go test ./...` — all pass (race detector runs in CI on Linux; the local
  Windows host has no C toolchain so `-race` cannot run here — documented).
- `govulncheck ./...` — 0 reachable vulnerabilities (re-run after fixes).
- `gosec ./...` — permission/error findings resolved; remaining hits are the
  by-design user-supplied paths (I2).
- Smoke: built the binary, added/found a snippet in a temp store inside the
  working dir, confirmed escape-sequence-laden content is stripped on `show` and
  in the store, and checked store file permissions.
