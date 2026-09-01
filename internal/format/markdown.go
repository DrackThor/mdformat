package format

import "strings"

// This file holds the line-level Markdown vocabulary the rules share: reading a
// line's prefix, recognizing the block it opens, and finding the spans that must
// stay verbatim. Document-level scanning (masks, fenced blocks) lives in
// [codeMask] and its neighbors in scan.go; anything only one rule needs stays
// with that rule.
//
// Every function here is covered by a table-driven test in markdown_test.go;
// the doc comments show one representative case each and name the test that
// holds the rest.

// indentedCodeWidth is the indentation at which Markdown treats content as an
// indented code block.
const indentedCodeWidth = 4

// splitIndent splits l into its leading whitespace and the rest of the line.
//
// For example, splitIndent("  \t- item") returns "  \t" and "- item"; a line
// that is only whitespace returns that whitespace and "".
//
// See TestSplitIndent.
func splitIndent(l string) (indent, rest string) {
	rest = strings.TrimLeft(l, " \t")
	return l[:len(l)-len(rest)], rest
}

// indentWidth returns the column width of a run of leading whitespace, counting
// a tab as [indentedCodeWidth] columns the way Markdown does.
//
// For example, indentWidth(" \t") returns 5: one space plus a four-column tab.
//
// See TestIndentWidth.
func indentWidth(indent string) int {
	n := 0
	for i := 0; i < len(indent); i++ {
		if indent[i] == '\t' {
			n += indentedCodeWidth
			continue
		}
		n++
	}
	return n
}

// leadingIndent returns the column width of a line's indentation.
//
// For example, leadingIndent("\t1. item") returns 4, while leadingIndent("x")
// returns 0.
//
// See TestLeadingIndent.
func leadingIndent(l string) int {
	indent, _ := splitIndent(l)
	return indentWidth(indent)
}

// scanPrefix measures a line's leading indentation and blockquote markers.
// quoteEnd is the index just past the last blockquote marker (0 when there is
// none), indent is the width of the whitespace between those markers and the
// content (a tab counts as four columns), and contentStart is the index of the
// first content byte.
//
// For example, scanPrefix("> >  text") returns 3, 2 and 5: two markers ending
// at index 3, two columns of indentation after them, and content from index 5.
// Whitespace before a marker belongs to the marker, not to indent.
//
// See TestScanPrefix.
func scanPrefix(line string) (quoteEnd, indent, contentStart int) {
	i := 0
	for {
		spaceStart := i
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i < len(line) && line[i] == '>' {
			i++
			quoteEnd = i
			continue
		}
		indent += indentWidth(line[spaceStart:i])
		return quoteEnd, indent, i
	}
}

// listMarkerLen returns the byte length of a leading list marker in s
// (bullet "- ", "* ", "+ " or ordered "1. " / "1) "), including its trailing
// whitespace, or 0 if s does not start with one.
//
// For example, listMarkerLen("12)  item") returns 5, while listMarkerLen("-x")
// returns 0: without whitespace after it, "-" is not a marker.
//
// See TestListMarkerLen.
func listMarkerLen(s string) int {
	if s == "" {
		return 0
	}
	var i int
	switch s[0] {
	case '-', '*', '+':
		i = 1
	default:
		i = orderedMarkerLen(s)
		if i == 0 {
			return 0
		}
	}
	spaces := 0
	for i+spaces < len(s) && (s[i+spaces] == ' ' || s[i+spaces] == '\t') {
		spaces++
	}
	if spaces == 0 {
		return 0
	}
	return i + spaces
}

// orderedMarkerLen returns the byte length of a leading ordered-list marker
// ("123." or "123)") in s, or 0 if s does not start with one.
//
// For example, orderedMarkerLen("42. item") returns 3 — the digits plus the
// delimiter, without the space that [listMarkerLen] also counts.
//
// See TestOrderedMarkerLen.
func orderedMarkerLen(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(s) || (s[i] != '.' && s[i] != ')') {
		return 0
	}
	return i + 1
}

// isThematicBreak reports whether l is a thematic break: three or more of the
// same "-", "*", or "_" character, optionally separated by spaces.
//
// For example, "- - -" and "___" are breaks; "--" is too short and "-*-" mixes
// characters, so neither is.
//
// See TestIsThematicBreak.
func isThematicBreak(l string) bool {
	s := strings.TrimSpace(l)
	if len(s) < 3 {
		return false
	}
	var ch byte
	count := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t':
			continue
		case '-', '*', '_':
			if ch == 0 {
				ch = s[i]
			} else if s[i] != ch {
				return false
			}
			count++
		default:
			return false
		}
	}
	return count >= 3
}

// fenceMarker returns the leading run of backticks or tildes (length >= 3) that
// starts s, or "" if s does not open/close a code fence.
//
// For example, fenceMarker("````go") returns "````" (the info string is left to
// the caller) and fenceMarker("~~x") returns "": a run shorter than three is
// not a fence.
//
// See TestFenceMarker.
func fenceMarker(s string) string {
	if !strings.HasPrefix(s, "```") && !strings.HasPrefix(s, "~~~") {
		return ""
	}
	return s[:runLen(s, 0, s[0])]
}

// parseATXHeading reports whether l is an ATX heading and returns its "#" run
// and the trimmed heading text (with any closing "#" sequence removed).
//
// For example, parseATXHeading("## Title ##") returns "##", "Title" and true,
// while parseATXHeading("#Title") returns false because CommonMark requires a
// space after the run.
//
// See TestParseATXHeading, which also covers the "# C#" case where a trailing
// "#" is part of the text rather than a closing sequence.
func parseATXHeading(l string) (hashes, text string, ok bool) {
	indent, s := splitIndent(l)
	if indentWidth(indent) > 3 { // indented that far = code, not a heading
		return "", "", false
	}
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return "", "", false
	}
	rest := s[n:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return "", "", false // "#text" is not a heading per CommonMark
	}
	text = strings.TrimSpace(rest)
	// Strip an optional closing run of '#' only when it is preceded by a space
	// (CommonMark), so "# C#" keeps its trailing '#'.
	if j := strings.LastIndexFunc(text, func(r rune) bool { return r != '#' }); j >= 0 {
		if text[j] == ' ' || text[j] == '\t' {
			text = strings.TrimRight(text[:j], " \t")
		}
	} else if text != "" { // text is all '#'
		text = ""
	}
	return s[:n], text, true
}

// isIndented reports whether a line's content is indented far enough to be an
// indented code block rather than wrapped prose.
//
// For example, isIndented("    code") is true, but isIndented("    - item") is
// false: a list marker makes the indentation nesting rather than code.
//
// ponytail: indentation alone, no block tracking. Costs the stitching of
// paragraphs nested four or more spaces deep inside a list; upgrade to a real
// indented-code mask in [codeMask] if that shows up.
//
// See TestIsIndented.
func isIndented(line string) bool {
	return classify(line) == kindIndentedCode
}

// inlineProtected marks byte positions in s that lie inside spans where breaks
// must not occur: inline code spans (backticks), autolinks (<...>), and link or
// image destinations (the "(...)" after "]").
//
// For example, in "see `a. b` and [x](y. z)" the positions covering "`a. b`"
// and "(y. z)" are marked, so the sentence splitter does not break at either
// period. The link text itself stays unmarked and may be broken. A code span
// closes only on a run as long as the one that opened it, so the interior
// backticks of "`a```b`" do not end it early.
//
// See TestInlineProtected.
func inlineProtected(s string) []bool {
	p := make([]bool, len(s))
	i := 0
	for i < len(s) {
		switch {
		case s[i] == '`':
			n := runLen(s, i, '`')
			if end := findRun(s, i+n, '`', n); end >= 0 {
				markRange(p, i, end+n)
				i = end + n
				continue
			}
		case s[i] == '<':
			if end := strings.IndexByte(s[i:], '>'); end >= 0 && !strings.ContainsAny(s[i:i+end], " ") {
				markRange(p, i, i+end+1)
				i = i + end + 1
				continue
			}
		case s[i] == ']' && i+1 < len(s) && s[i+1] == '(':
			if end := strings.IndexByte(s[i+1:], ')'); end >= 0 {
				markRange(p, i+1, i+1+end+1)
				i = i + 1 + end + 1
				continue
			}
		}
		i++
	}
	return p
}

// runLen returns the length of the run of ch bytes starting at index i, which
// is 0 when s[i] is not ch.
//
// For example, runLen("~~~~go", 0, '~') returns 4.
//
// See TestRunLen.
func runLen(s string, i int, ch byte) int {
	n := 0
	for i+n < len(s) && s[i+n] == ch {
		n++
	}
	return n
}

// findRun returns the start index of the first maximal run of exactly n ch
// bytes at or after from, or -1.
//
// The run must be maximal on both sides, so a shorter run is never matched
// inside a longer one: searching for a lone backtick past the opening one of
// "`a```b`" skips the interior run of three and finds the final backtick, which
// is where the code span really ends. That is what [inlineProtected] needs from
// a closing delimiter.
//
// See TestFindRun.
func findRun(s string, from int, ch byte, n int) int {
	for i := from; i < len(s); i++ {
		if s[i] != ch || (i > 0 && s[i-1] == ch) {
			continue
		}
		if runLen(s, i, ch) == n {
			return i
		}
	}
	return -1
}

// markRange sets p[lo:hi] to true, stopping at the end of p.
//
// For example, markRange(make([]bool, 3), 1, 9) marks indexes 1 and 2 and
// ignores the rest of the range.
//
// See TestMarkRange.
func markRange(p []bool, lo, hi int) {
	for i := lo; i < hi && i < len(p); i++ {
		p[i] = true
	}
}

// lineKind is the block a line opens, as the rules need to tell them apart.
type lineKind int

const (
	kindBlank lineKind = iota
	kindATXHeading
	kindSetextUnderline
	kindFence
	kindThematicBreak
	kindListItem
	kindTableRow
	kindHTMLBlock
	kindIndentedCode
	kindParagraph
)

// classify reports which block a line opens. It looks at the content after the
// line's indentation and blockquote markers, so quoted prose classifies as
// prose; callers that must not treat quoted content as top level pair this with
// [quoteDepth].
//
// A "---" run classifies as [kindThematicBreak]: whether it is instead a Setext
// underline depends on the line before it, which only setext.go knows.
//
// For example, classify("> # Title") is [kindATXHeading], classify("    code")
// is [kindIndentedCode], and classify("prose with a | pipe") is
// [kindTableRow] — the table check is deliberately crude.
//
// See TestClassify in format_test.go for the full matrix.
func classify(l string) lineKind {
	if strings.TrimSpace(l) == "" {
		return kindBlank
	}
	_, indent, contentStart := scanPrefix(l)
	content := l[contentStart:]
	switch {
	case content == "": // a blockquote marker with nothing after it
		return kindBlank
	case indent >= indentedCodeWidth && listMarkerLen(content) == 0:
		return kindIndentedCode
	case isThematicBreak(content):
		return kindThematicBreak
	case setextLevel(content) == 1: // a "=" run; "-" runs are thematic breaks
		return kindSetextUnderline
	case fenceMarker(content) != "":
		return kindFence
	case listMarkerLen(content) > 0:
		return kindListItem
	case looksLikeRow(l):
		return kindTableRow
	case content[0] == '<':
		return kindHTMLBlock
	}
	if _, _, ok := parseATXHeading(content); ok {
		return kindATXHeading
	}
	return kindParagraph
}
