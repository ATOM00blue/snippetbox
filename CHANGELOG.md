# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

- **Terminal escape-sequence injection** — snippet content, titles, languages and
  tags are now stripped of ANSI/OSC/control sequences before being rendered in
  the TUI preview/list or printed by the `show`/`list`/`find` subcommands. A
  malicious snippet can no longer clear your screen, rewrite the terminal title,
  hide text, or inject clipboard/response sequences. (CWE-150)
- **Secrets at rest** — the store file is now created `0600` (was relying on a
  temp-file default) and its directory `0700`; `export -o` writes `0600` (was
  `0644`). Snippets may contain tokens/secrets, so they are owner-only.
- **Untrusted import hardening** — `import` now caps the input at 16 MiB
  (memory-exhaustion DoS), validates and sanitizes every imported snippet
  (control characters stripped, fields length-capped, empty title/content
  rejected and skipped), and rejects JSON that is neither a snippet array nor a
  store document instead of silently importing nothing.

### Changed

- All snippet writes (`add`, `edit`, `import`) now sanitize and validate fields,
  so control characters can never be persisted to the store.

## [0.1.0] - 2026-05-22

### Added

- Interactive Bubble Tea TUI: filterable snippet list, syntax-highlighted
  preview pane, and a status/help bar.
- Fuzzy search across snippet **title, tags, language, and content**.
- One-key copy to the system clipboard (`c` in the TUI).
- In-TUI add / edit / delete with a multi-field form (`a` / `e` / `d`).
- Scriptable subcommands: `add` (flags or `--stdin`), `find` (`--json`, `-n`),
  `copy`, `show`, `list` (`--json`), `delete`, `import`, `export`, `version`.
- Single-file JSON store in the OS config dir, with atomic writes.
- `--store` flag and `SNIPPETBOX_STORE` env var to override the store path.
- Cross-platform support (Linux, macOS, Windows).
- Test suite covering the snippet model, store, search, and CLI; CI on Linux,
  macOS, and Windows.

[Unreleased]: https://github.com/ATOM00blue/snippetbox/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ATOM00blue/snippetbox/releases/tag/v0.1.0
