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
| `setext-headings`       | Rewrites Setext (underlined) headings as ATX, joining a hard-wrapped heading into one line. |
| `atx-headings`          | Normalizes `#` spacing and strips optional closing `#` sequences.                       |
| `list-markers`          | Normalizes unordered bullets to a single marker (`-` by default).                       |
| `ordered-list-numbering` | Renumbers ordered lists consecutively, keeping the number each list starts at (`increment` or `keep`). |
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
  - setext-headings
  - atx-headings
  - list-markers
  - ordered-list-numbering
  - semantic-line-breaks
  - table-width
  - blank-lines

options:
  semantic-line-breaks:
    break-on: [sentence] # add: colon, semicolon, em-dash, comma
  ordered-list-numbering:
    style: increment # or: keep
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

Every rule ships with a golden fixture pair named after it —
`test-cases/inputs/<rule-name>.md` and `test-cases/expected/<rule-name>.md`.
Write the input by hand so it covers the rule and its edge cases,
then generate the expected output and read it before committing:

```bash
go run ./scripts/gen_expected.go
```

Fixtures are formatted with the full default rule set and are also asserted to be idempotent,
so they catch interactions between rules as well as the rule under test.

## Development

```bash
make build   # build the binary
make test    # go test -race ./...
make lint    # golangci-lint
```

Golden fixtures live in `test-cases/`, paired up by `test-cases/test-config.yaml`:

```yaml
cases:
  - input: ordered-list-numbering.md
    expected: ordered-list-numbering-keep.md
    config: # optional, same shape as .mdformat.yaml
      options:
        ordered-list-numbering:
          style: keep
```

Each case formats `inputs/<input>` and compares the result to `expected/<expected>`,
then formats it once more to assert the output is idempotent.
The optional `config` lets one input have a golden per option value.

Write the expected files by hand.
They state what the formatter *should* do, so generating them by running the formatter would
only ever confirm what it already does.
