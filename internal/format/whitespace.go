package format

import (
	"strings"

	"github.com/spf13/viper"
)

const nameBlankLines = "blank-lines"

func init() { Register(nameBlankLines, newBlankLines) }

// blankLines normalizes vertical whitespace: it collapses runs of blank lines to
// at most maxBlank (default 1 blank line = two newlines), ensures a blank line
// before and after ATX headings and fenced code blocks, and strips leading and
// trailing blank lines. Blank lines inside verbatim spans are preserved.
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

// surroundBlocks inserts a single blank line before and after ATX headings and
// fenced code blocks when adjacent to non-blank content.
func surroundBlocks(lines []string) []string {
	mask := codeMask(lines)
	out := make([]string, 0, len(lines))

	for i := range lines {
		wantBefore := isHeadingLine(lines[i], mask[i]) || isFenceEdge(lines, mask, i, -1)
		if wantBefore && len(out) > 0 && !isBlankLine(out[len(out)-1]) {
			out = append(out, "")
		}
		out = append(out, lines[i])

		wantAfter := isHeadingLine(lines[i], mask[i]) || isFenceEdge(lines, mask, i, +1)
		if wantAfter && i+1 < len(lines) && !isBlankLine(lines[i+1]) {
			out = append(out, "")
		}
	}
	return out
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
