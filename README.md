# mdformat

A command-line tool written in Go that formats Markdown files in place using a configurable, ordered set of formatting rules.

Rules follow the naming and configuration conventions of established formatters like Prettier and Black: each rule has a kebab-case name, can be enabled or disabled individually, and runs in a configured order.
New rules plug in through a single standardized interface.

## Features

- Format a **single file** or **every Markdown file in a folder** (recursively with `-r`).
- Always formats **in place**.
- **Modular rules** applied in a configured order; enable or disable each one.
- **Config file** driven, with layered precedence (project config overrides user config).
- `--check` mode reports files that need formatting without writing them.

## Rules

| Rule                      | What it does                                                                                                          |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `trailing-whitespace`     | Trims trailing spaces/tabs; preserves Markdown hard line breaks (two trailing spaces).                                |
| `hard-tabs`               | Expands hard tabs to the next tab stop (width 4 by default); tabs in code keep their tab.                             |
| `code-fence-style`        | Rewrites fenced code blocks with one marker (backticks by default) and the shortest valid fence length.               |
| `setext-headings`         | Rewrites Setext (underlined) headings as ATX, joining a hard-wrapped heading into one line.                           |
| `atx-headings`            | Normalizes `#` spacing, strips optional closing `#` sequences and trailing punctuation (`.,;:!?`).                    |
| `list-markers`            | Normalizes unordered bullets to a single marker (`-` by default).                                                     |
| `ordered-list-numbering`  | Renumbers ordered lists consecutively, keeping the number each list starts at (`increment` or `keep`).                |
| `semantic-line-breaks`    | Unwraps hard-wrapped paragraphs, then puts each sentence (and optionally each clause) on its own line ([sembr.org]).  |
| `table-width`             | Pads GFM table cells to the column's widest content plus a padding (default 1).                                       |
| `blank-lines`             | Collapses excess blank lines and ensures blanks around headings, code fences, and lists.                              |

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
  - hard-tabs
  - code-fence-style
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
  atx-headings:
    strip-trailing-punctuation: ".,;:!?" # "" keeps heading punctuation
  code-fence-style:
    marker: "`" # or: ~
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
| `--verbose`        | `-v`   | Name the rules that changed each file; `-vv` adds the lines.     |
| `--version`        |        | Print the mdformat version and exit.                             |

## Verbose output

`-v` names the rules that changed a file, `-vv` adds the lines each rule touched.
Both write to stderr, so stdout keeps the `formatted <file>` lines a pipeline reads.
The same setting lives in the config as `verbosity: 0|1|2`; the flag wins when both are given.

```console
$ mdformat -v README.md
README.md: table-width, blank-lines
formatted README.md

$ mdformat -vv README.md
README.md  table-width       lines 21-28
README.md  blank-lines       lines 19, 97
formatted README.md
```

Line numbers are as of the rule that reported them.
A rule running later that inserts or removes lines shifts the ones before it.
A rule that changes the line count reports the region it rewrote rather than single lines.

## Adding a rule

Implement the `Rule` interface in `internal/format`, register it in an `init` function, and add it to the default order:

```go
type Rule interface {
    Name() string
    Apply(lines []string) ([]string, error)
}
```

Every rule ships with a golden fixture: an input under `test-cases/inputs/`, the expected output under `test-cases/expected/`, and a case pairing them in `test-cases/test-config.yaml`.

## Development

```bash
make build             # build the binary
make test              # go test -race ./...
make test-integration  # golden-file tests in test-cases/ only
make lint              # golangci-lint
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

Each case formats `inputs/<input>` and compares the result to `expected/<expected>`, then formats it once more to assert the output is idempotent.
The optional `config` lets one input have a golden per option value.

Write the expected files by hand.
They state what the formatter *should* do, so generating them by running the formatter would only ever confirm what it already does.
