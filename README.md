# mdformat

A command-line tool written in Go that formats Markdown files in place using a configurable, ordered set of formatting rules.

Rules follow the naming and configuration conventions of established formatters like Prettier and Black:
each rule has a kebab-case name, can be enabled or disabled individually, and runs in a configured order.
New rules plug in through a single standardized interface.

## Features

- Format a **single file** or **every Markdown file in a folder** (recursively with `-r`).
- Always formats **in place**.
- **Modular rules** applied in a configured order; enable or disable each one.
- **Config file** driven, with layered precedence (project config overrides user config).
- `--check` mode reports files that need formatting without writing them.

## Rules

| Rule                    | What it does                                                                            |
| ----------------------- | --------------------------------------------------------------------------------------- |
| `trailing-whitespace`   | Trims trailing spaces/tabs; preserves Markdown hard line breaks (two trailing spaces).  |
| `atx-headings`          | Normalizes `#` spacing and strips optional closing `#` sequences.                       |
| `list-markers`          | Normalizes unordered bullets to a single marker (`-` by default).                       |
| `semantic-line-breaks`  | Unwraps hard-wrapped paragraphs, then puts each sentence (and optionally each clause) on its own line ([sembr.org]). |
| `table-width`           | Pads GFM table cells to the column's widest content plus a padding (default 1).         |
| `blank-lines`           | Collapses excess blank lines and ensures blanks around headings and code fences.        |

Verbatim spans — fenced code blocks, inline code, front matter, and link/image destinations — are never altered.

[sembr.org]: https://sembr.org

## Install

### From source

```bash
git clone https://github.com/drackthor/mdformat.git
cd mdformat
go build -o mdformat .
```

### go install

```bash
go install github.com/drackthor/mdformat@latest
```

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/drackthor/mdformat/main/install.sh | sh
```

## Usage

Format a single file in place:

```bash
mdformat README.md
```

Format every Markdown file in a folder (top level only):

```bash
mdformat docs/
```

Recurse into subdirectories:

```bash
mdformat -r docs/
```

Check without writing (exits non-zero if any file would change):

```bash
mdformat --check docs/
```

## Configuration

Configuration is layered.
Each layer overrides the one before it:

1. Built-in defaults.
2. User config: `$XDG_CONFIG_HOME/mdformat/config.yaml` (falls back to `~/.config/mdformat/config.yaml`).
3. Project config in the current directory: `./.mdformat.yaml`.
4. An explicit `--config <file>`.

The `rules` list defines both which rules run and the order they run in.
Remove a rule from the list to disable it.
See [.mdformat.example.yaml](.mdformat.example.yaml) for all options.

```yaml
rules:
  - trailing-whitespace
  - atx-headings
  - list-markers
  - semantic-line-breaks
  - table-width
  - blank-lines

options:
  semantic-line-breaks:
    break-on: [sentence] # add: colon, semicolon, em-dash, comma
  table-width:
    padding: 1
```

## Flags

| Flag               | Short  | Description                                                      |
| ------------------ | ------ | ---------------------------------------------------------------- |
| `--config <file>`  | `-c`   | Explicit config file (highest precedence).                       |
| `--recursive`      | `-r`   | Recurse into directories.                                        |
| `--check`          |        | Report files needing formatting without writing; exit non-zero.  |
| `--version`        |        | Print the mdformat version and exit.                             |

## Adding a rule

Implement the `Rule` interface in `internal/format`, register it in an `init` function, and add it to the default order:

```go
type Rule interface {
    Name() string
    Apply(lines []string) ([]string, error)
}
```

## Development

```bash
make build   # build the binary
make test    # go test -race ./...
make lint    # golangci-lint
```

Regenerate golden test fixtures after changing rule behavior:

```bash
go run ./scripts/gen_expected.go
```
