# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
