package format

import (
	"strings"

	"github.com/spf13/viper"
)

const nameBlankLines = "blank-lines"

func init() { Register(nameBlankLines, newBlankLines) }

// blankLines normalizes vertical whitespace: it collapses runs of blank lines to
// at most maxBlank (default 1 blank line = two newlines), ensures a blank line
// before and after ATX headings, fenced code blocks and lists, and strips
// leading and trailing blank lines. Blank lines inside verbatim spans are
// preserved.
type blankLines struct {
	maxBlank int
}

func newBlankLines(opts *viper.Viper) (Rule, error) {
	m := 1
	if opts != nil && opts.IsSet("max-blank") {
		m = opts.GetInt("max-blank")
		if m < 1 {
			m = 1
		}
	}
	return blankLines{maxBlank: m}, nil
}

func (blankLines) Name() string { return nameBlankLines }

func (r blankLines) Apply(lines []string) ([]string, error) {
	lines = r.collapse(lines)
	lines = surroundBlocks(lines)
	return trimEdges(lines), nil
}

func (r blankLines) collapse(lines []string) []string {
	mask := codeMask(lines)
	out := make([]string, 0, len(lines))
	run := 0
	for i, l := range lines {
		if mask[i] {
			out = append(out, l)
			run = 0
			continue
		}
		if strings.TrimSpace(l) == "" {
			run++
			if run <= r.maxBlank {
				out = append(out, "")
			}
			continue
		}
		run = 0
		out = append(out, l)
	}
	return out
}

// surroundBlocks inserts a single blank line before and after ATX headings,
// fenced code blocks and lists when adjacent to non-blank content.
func surroundBlocks(lines []string) []string {
	mask := codeMask(lines)
	starts, inList := listEdges(lines, mask)
	out := make([]string, 0, len(lines))

	for i := range lines {
		wantBefore := isHeadingLine(lines[i], mask[i]) || isFenceEdge(lines, mask, i, -1) || starts[i]
		if wantBefore && len(out) > 0 && !isBlankLine(out[len(out)-1]) {
			out = append(out, "")
		}
		out = append(out, lines[i])

		wantAfter := isHeadingLine(lines[i], mask[i]) || isFenceEdge(lines, mask, i, +1) ||
			(inList[i] && i+1 < len(lines) && !mask[i+1] && endsList(lines[i+1]))
		if wantAfter && i+1 < len(lines) && !isBlankLine(lines[i+1]) {
			out = append(out, "")
		}
	}
	return out
}

// listEdges reports, per line, whether it opens a list block that a blank line
// may be inserted in front of (starts), and whether it belongs to a list block
// at all (inList).
//
// Only starts that can interrupt a paragraph are reported, so a blank line is
// never inserted where it would turn prose into a list (markdownlint MD032 is
// not applied when the fix would change what the document renders). For the same
// reason a blank line is never inserted before a lazy continuation: the text
// under a list item belongs to that item, so splitting it off would change the
// render rather than tidy it.
func listEdges(lines []string, mask []bool) (starts, inList []bool) {
	starts = make([]bool, len(lines))
	inList = make([]bool, len(lines))

	open := false
	blank := true // the top of the document reads like a preceding blank line
	for i, l := range lines {
		switch {
		case mask[i]:
			// Fenced content and front matter keep an open list open; a fence
			// that actually ends the list is already surrounded as a fence.
			inList[i] = open
			blank = false
		case isBlankLine(l):
			blank = true
		default:
			indent, rest := splitIndent(l)
			switch {
			case len(indent) < indentedCodeWidth && listMarkerLen(rest) > 0 && !isThematicBreak(l):
				if !open {
					starts[i] = canInterruptParagraph(rest)
				}
				open = true
			case open && (!blank || indent != ""):
				// A lazy continuation (directly under item content) or an
				// indented block still belongs to the item.
			default:
				open = false
			}
			inList[i] = open
			blank = false
		}
	}
	return starts, inList
}

// canInterruptParagraph reports whether a list item written as rest starts a
// list on the line right after a paragraph: the item needs content, and an
// ordered item must start at 1 (CommonMark).
func canInterruptParagraph(rest string) bool {
	marker := listMarkerLen(rest)
	if marker == 0 || strings.TrimSpace(rest[marker:]) == "" {
		return false
	}
	if n := orderedMarkerLen(rest); n > 0 {
		return rest[:n-1] == "1"
	}
	return true
}

// endsList reports whether l closes an open list block instead of continuing it
// lazily. Headings and fences do too, but they are surrounded on their own.
func endsList(l string) bool {
	indent, rest := splitIndent(l)
	if len(indent) >= indentedCodeWidth {
		return false
	}
	return strings.HasPrefix(rest, ">") || isThematicBreak(l)
}

func isBlankLine(s string) bool { return strings.TrimSpace(s) == "" }

func isHeadingLine(l string, masked bool) bool {
	if masked {
		return false
	}
	_, _, ok := parseATXHeading(l)
	return ok
}

// isFenceEdge reports whether line i is a fence marker at the boundary of a
// fenced block: dir -1 checks the opening line, dir +1 the closing line.
func isFenceEdge(lines []string, mask []bool, i, dir int) bool {
	if !mask[i] || fenceMarker(strings.TrimLeft(lines[i], " \t")) == "" {
		return false
	}
	neighbor := i + dir
	return neighbor < 0 || neighbor >= len(lines) || !mask[neighbor]
}

func trimEdges(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}
