# mdformat — Implementation Backlog

Feature parity targets and future work, derived from:

- [hukkin/mdformat](https://github.com/hukkin/mdformat) (Python, opinionated CommonMark formatter) — features we lack.
- [markdownlint](https://github.com/DavidAnson/markdownlint) rules commonly enforced via pre-commit.
- [psf/black](https://github.com/psf/black) philosophy and CLI ergonomics.

Sources:

- [mdformat style guide](https://mdformat.readthedocs.io/en/stable/users/style.html)
- [mdformat configuration](https://mdformat.readthedocs.io/en/stable/users/configuration_file.html)
- [markdownlint rules](https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md)

## How to implement a new rule

Every item tagged **[rule]** is a new `Rule` in `internal/format`.
The contract is already in place — follow `AGENTS.md` and the existing rules:

```go
// internal/format/<rule>.go
const nameMyRule = "my-rule"

func init() { Register(nameMyRule, newMyRule) }

type myRule struct{ opt int }

func newMyRule(opts *viper.Viper) (Rule, error) {
    r := myRule{opt: 2} // defaults live in the constructor (viper.Sub loses defaults)
    if opts != nil && opts.IsSet("opt") {
        r.opt = opts.GetInt("opt")
    }
    return r, nil
}

func (myRule) Name() string { return nameMyRule }

func (r myRule) Apply(lines []string) ([]string, error) {
    mask := codeMask(lines) // ALWAYS skip verbatim spans (fences, front matter)
    // ... transform lines[i] where !mask[i] ...
    return lines, nil
}
```

Then:

1. Add `nameMyRule` to `DefaultRuleOrder` in `rule.go` at the right position.
2. Document the option in `.mdformat.example.yaml`.
3. Add a table-driven case in `format_test.go` and, if useful, a fixture under `test-cases/`.
4. Keep it idempotent — the integration test formats twice and asserts equality.

Items tagged **[flag]** are CLI/config plumbing in `cmd/root.go` and `internal/config`.
Items tagged **[core]** touch the engine or scanner.

______________________________________________________________________

## Tier 1 — high value, low risk

### 1. `setext-headings` → convert to ATX [rule] — DONE

Implemented in `internal/format/setext.go`, runs before `atx-headings` in `DefaultRuleOrder`.

mdformat converts Setext (underline) headings to ATX for consistency (markdownlint MD003).
We currently only normalize existing ATX headings.

```markdown
Title
=====

Subtitle
--------
```

becomes

```markdown
# Title

## Subtitle
```

Implementation: detect a non-blank text line followed by a line of only `=` (H1) or `-` (H2),
outside `codeMask`, and where the underline is not a thematic break / table delimiter.
Emit `# `/`## ` + text, drop the underline.
Note ordering: must run before `blank-lines` so the surround pass sees a real heading.

### 2. `ordered-list-numbering` [rule] — DONE

Implemented in `internal/format/ordered.go`, runs after `list-markers` in `DefaultRuleOrder`.
The number a list starts at is preserved (a list written as `3.` still renders as 3);
only the items after the first are rewritten.

mdformat renumbers ordered lists and defaults to the non-incrementing `1.` style,
which minimizes diffs when items are inserted (markdownlint MD029).

Option `style`:

- `increment`: `1.`, `2.`, `3.`, …
- `keep`: leave as-is

```markdown
1. first
3. second
5. third
```

with `style: increment`:

```markdown
1. first
2. second
3. third
```

with `style: keep`:

```markdown
1. first
3. second
5. third
```

Implementation: track list blocks (leading indent + `orderedMarkerLen` already exists in `sembr.go`).
Reuse `listMarkerLen`/`orderedMarkerLen`. Reset numbering per list (blank line or dedent ends a list).

### 3. `hard-tabs` → spaces [rule]

markdownlint MD010. Expand leading hard tabs to spaces (configurable width, default 4).
Must NOT touch tabs inside fenced/inline code.

```markdown
-\tItem with a tab
```

becomes

```markdown
-   Item with a tab
```

Implementation: replace `\t` outside `codeMask` and outside inline-code spans
(reuse `inlineProtected` from `sembr.go`).

### 4. `heading-trailing-punctuation` [rule]

markdownlint MD026. Strip trailing `.,;:!?` from heading text (configurable punctuation set).

```markdown
## Overview:
```

becomes

```markdown
## Overview
```

Implementation: small extension; could fold into `atx-headings` as an option
`strip-trailing-punctuation: ".,;:!?"` rather than a separate rule.

### 5. `code-fence-style` [rule]

markdownlint MD048 + MD046 (fenced over indented).
Normalize all fences to one marker (default backticks) and normalize fence length.

Input uses `~~~` fences; output uses backtick fences (shown with tilde wrappers here to
avoid nested code fences):

~~~text
before:  ~~~ … ~~~
after:   ``` … ```
~~~

Two parts:

- **fence marker**: rewrite `~~~` ↔ backtick fences (respect content that contains the target marker — widen instead).
- **indented → fenced** (MD046): convert 4-space indented code blocks to fenced. Higher effort; keep as its own sub-task.

`scan.go`'s `fenceMarker` already parses both markers — extend the scanner to report the block so the rule can rewrite the edges.

### 6. `blank-lines` — surround lists (MD032) [core]

Extend the existing `blank-lines` rule to also ensure a blank line before and after
list blocks, not just headings and fences.

```markdown
text
- a
- b
next
```

becomes

```markdown
text

- a
- b

next
```

Implementation: `surroundBlocks` in `whitespace.go` — add a list-block edge detector alongside
`isHeadingLine`/`isFenceEdge`. Careful with tight vs. loose lists.

______________________________________________________________________

## Tier 2 — inline normalization (needs an inline tokenizer)

These need reliable inline parsing so we never corrupt code spans, links, or escapes.
Reuse and extend `inlineProtected` (backticks, autolinks, link destinations) into a small
inline tokenizer shared by all inline rules.

### 7. `emphasis-style` [rule]

markdownlint MD004 (bullets, above) has an inline analog: normalize emphasis markers.
mdformat uses `_` for italic and `**` for bold by default (configurable).

```markdown
*italic* and __bold__
```

becomes

```markdown
_italic_ and **bold**
```

Option `italic: "_"|"*"`, `bold: "**"|"__"`.
Do NOT rewrite markers that would change meaning (e.g. intra-word `_` in snake_case must stay literal).

### 8. `code-span-trim` [rule]

markdownlint MD038. Trim a single leading/trailing space inside code spans
and minimize backtick fence length to the shortest that is valid.

```markdown
` code ` and ``a`b``
```

becomes

```markdown
`code` and ``a`b``
```

### 9. `link-normalization` [rule]

- Remove redundant angle brackets around destinations: `[t](<url>)` → `[t](url)` (mdformat).
- markdownlint MD039: trim spaces inside link text `[ text ](u)` → `[text](u)`.
- markdownlint MD034: wrap bare URLs as autolinks `http://x` → `<http://x>`.

```markdown
See [ the docs ](<https://example.com>) and http://bare.example.com
```

becomes

```markdown
See [the docs](https://example.com) and <http://bare.example.com>
```

### 10. `reference-links` [rule]

mdformat moves all link reference definitions to the bottom, sorted by label,
and drops duplicate/unused definitions (markdownlint MD052/MD053).

```markdown
[a]: http://a.example.com
Text [a][a] and [b][b].
[b]: http://b.example.com
[unused]: http://x.example.com
```

becomes

```markdown
Text [a][a] and [b][b].

[a]: http://a.example.com
[b]: http://b.example.com
```

Higher effort: needs a document-wide pass, not just per-line. Consider a `Rule` variant that
sees the whole doc (current interface already passes all lines).

### 11. `hard-line-break` policy [rule / core]

mdformat replaces the invisible two-trailing-spaces hard break with a trailing backslash.
Our `trailing-whitespace` rule currently PRESERVES two trailing spaces.
Make this an explicit choice.

Option `hard-break: "backslash"|"spaces"|"keep"`:

```markdown
line one·· (two spaces)
line two
```

with `hard-break: backslash`:

```markdown
line one\
line two
```

______________________________________________________________________

## Tier 3 — CLI & engine ergonomics (black / mdformat parity)

### 12. `--wrap {keep,no,N}` [flag/core]

mdformat's paragraph wrapping. `keep` (default), `no` (unwrap to one line per paragraph),
or an integer column to reflow to. This generalizes our `semantic-line-breaks`;
consider whether `wrap` and `semantic-line-breaks` are two modes of one wrapping engine.

### 13. `--end-of-line {lf,crlf,keep}` [flag/core]

We hard-code LF in `engine.go`. Make it configurable (default `lf`), and support `keep`
(detect the file's dominant ending and preserve it). mdformat parity.

### 14. `--diff` and stdin/stdout [flag] (black + mdformat)

- `-` as a path → read stdin, write formatted result to stdout (pipeline use).
- `--diff` → print a unified diff instead of writing; pairs with `--check`.
- `--quiet` / `--color` for CI ergonomics.

`--check` already exists and returns non-zero when files would change (black/mdformat semantics).

### 15. `--exclude` glob patterns [flag] (black + mdformat)

Skip files matching Unix globs; also add `.gitignore`-aware discovery.
Directory walking already lives in `collectFiles`/`walkDir` in `cmd/root.go`.

```bash
mdformat -r . --exclude 'vendor/**' --exclude 'CHANGELOG.md'
```

### 16. TOML config parity [flag]

mdformat uses `.mdformat.toml`; black uses `[tool.black]` in `pyproject.toml`.
Viper already supports TOML — accept `.mdformat.toml` in addition to `.mdformat.yaml`
in `internal/config` (same precedence chain). Low effort, nice for cross-tool users.

### 17. Safety / idempotency guarantee [core] (black's headline feature)

black verifies the reformatted output parses to an equivalent AST; mdformat re-renders to HTML
and asserts the render is unchanged (`--validate`, on by default).

- Minimum: after formatting, run the engine again and assert output is stable; warn/fail on drift.
- Stretch: render input and output Markdown to HTML and compare (a `--validate` / `--no-validate` flag).
  This catches any rule that accidentally changes meaning.
  The integration test already checks idempotency — promote it to a runtime guard.

### 18. Per-run rule overrides on the CLI [flag]

black is config-light but predictable; expose quick overrides without editing config:

```bash
mdformat --enable table-width --disable semantic-line-breaks README.md
```

Maps onto the existing ordered `rules` list built in `format.Build`.

______________________________________________________________________

## Tier 4 — parser extensions (GFM & friends)

mdformat ships these as plugins; we can add them as rules or as a plugin layer.
Our `Register` registry is already the seam for an extension system.

- **GFM task lists**: normalize `- [ ]` / `- [x]` spacing.
- **Strikethrough / autolinks**: leave content intact; ensure inline rules skip them.
- **Footnotes** (`mdformat-footnote`): normalize `[^1]` definitions, move to bottom, renumber.
- **Front matter** (`mdformat-frontmatter`): we already treat leading `---` as verbatim (`scan.go`);
  optional: format the YAML/TOML inside.
- **TOC generation** (`mdformat-toc`): populate a `<!-- toc -->` marker from headings.
- **Table column count / pipe style** (markdownlint MD055/MD056): our `table-width` should
  also pad ragged rows to the header's column count and enforce leading/trailing pipes consistently.

______________________________________________________________________

## Lint-only checks (report, do not autofix)

Some markdownlint rules cannot be safely auto-fixed — surface them under a `--lint` mode
(exit non-zero, print `file:line rule message`) instead of rewriting:

| Rule  | Check                                            | Why not auto-fix                    |
| ----- | ------------------------------------------------ | ----------------------------------- |
| MD001 | Heading levels increment by one                  | Correct level is author intent      |
| MD025 | Only one top-level (H1) heading                  | Which H1 to demote is ambiguous     |
| MD040 | Fenced code blocks specify a language            | Cannot infer the language           |
| MD036 | Emphasis used instead of a heading               | Intent is ambiguous                 |
| MD005 | Consistent list-item indentation at same level   | Fix may change nesting              |

______________________________________________________________________

## Suggested order of attack

1. Tier 1 (all six; #1 and #2 done) — pure line transforms, reuse existing helpers, high user value.
2. #16 TOML config + #13 end-of-line + #14 diff/stdin — cheap CLI parity wins.
3. Build the shared inline tokenizer, then Tier 2 (#7–#11).
4. #17 safety guard once several inline rules exist (highest risk of meaning changes).
5. Tier 3 wrap engine (#12) and Tier 4 extensions as needed.
