# Contributing to snippetbox

Thanks for your interest in improving snippetbox! This document explains how to
get set up and what we expect from contributions.

## Getting started

You'll need [Go 1.24+](https://go.dev/dl/).

```sh
git clone https://github.com/ATOM00blue/snippetbox.git
cd snippetbox
go build -o snippetbox .
./snippetbox
```

Run against an isolated store while hacking so you don't touch your real
snippets:

```sh
SNIPPETBOX_STORE=./dev-store.json ./snippetbox
```

## Project layout

```
main.go                  thin entry point -> cmd.Execute
cmd/                     CLI parsing + scriptable subcommands
internal/snippet/        the Snippet model + helpers
internal/store/          JSON persistence (atomic writes, CRUD)
internal/search/         fuzzy search over snippets
internal/tui/            Bubble Tea TUI (list + preview + form + highlight)
```

- Keep storage/search logic free of TUI concerns.
- The TUI imports `store` and `search`; never the other way around.

## Before you open a PR

Please make sure all of these pass:

```sh
gofmt -l .        # should print nothing
go vet ./...      # should be clean
go test ./...     # should be green
```

Guidelines:

- **Add tests** for new behavior. Storage/search/cmd logic should be covered by
  table-driven tests using a temp store (see existing `*_test.go`).
- **Keep it cross-platform.** Avoid OS-specific paths; use `path/filepath` and
  `os.UserConfigDir`. Don't break Windows.
- **Small, focused commits** with clear messages.
- Document any new keybinding or subcommand in the README.

## Reporting bugs / requesting features

Open an issue with:

- what you expected vs. what happened,
- steps to reproduce (and your OS + `snippetbox version`),
- for features, the use case you're trying to solve.

## Code of conduct

Be kind and constructive. We want snippetbox to be a welcoming project.
