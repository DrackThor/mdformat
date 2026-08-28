package format

import (
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const (
	nameOrderedListNumbering = "ordered-list-numbering"
	orderedStyleIncrement    = "increment"
	orderedStyleKeep         = "keep"
)

func init() { Register(nameOrderedListNumbering, newOrderedListNumbering) }

// orderedListNumbering renumbers ordered list items (markdownlint MD029). The
// number a list starts at is always kept — a list written as "3." still renders
// as 3. — and only the items after it are rewritten:
//
//	increment  (default) the starting number counts up: 1. 2. 3.
//	keep       leave every marker as written
//
// Nested lists are numbered independently, and a list ends where Markdown ends
// it: at a dedent, at a bullet item taking over the same level, or at a
// top-level block after a blank line.
type orderedListNumbering struct {
	style string
}

func newOrderedListNumbering(opts *viper.Viper) (Rule, error) {
	r := orderedListNumbering{style: orderedStyleIncrement}
	if opts != nil && opts.IsSet("style") {
		switch s := strings.ToLower(strings.TrimSpace(opts.GetString("style"))); s {
		case orderedStyleIncrement, orderedStyleKeep:
			r.style = s
		}
	}
	return r, nil
}

func (orderedListNumbering) Name() string { return nameOrderedListNumbering }

// orderedLevel is one nesting level of ordered list being renumbered.
type orderedLevel struct {
	indent int  // indentation of the item markers at this level
	delim  byte // '.' or ')'; switching delimiter starts a new list
	num    int  // number to write on the next item
}

func (r orderedListNumbering) Apply(lines []string) ([]string, error) {
	if r.style == orderedStyleKeep {
		return lines, nil
	}
	mask := codeMask(lines)
	out := make([]string, len(lines))
	var levels []orderedLevel
	sawBlank := false

	for i, l := range lines {
		switch {
		case mask[i]:
			out[i] = l
		case isBlankLine(l):
			out[i] = l
			sawBlank = true
		default:
			out[i] = r.applyLine(l, sawBlank, &levels)
			sawBlank = false
		}
	}
	return out, nil
}

// applyLine handles one non-blank line: it renumbers an ordered item, or closes the
// list levels the line brings to an end, and returns the line to write.
func (r orderedListNumbering) applyLine(l string, afterBlank bool, levels *[]orderedLevel) string {
	indent := leadingIndent(l)
	// A dedent closes every deeper list.
	for len(*levels) > 0 && (*levels)[len(*levels)-1].indent > indent {
		*levels = (*levels)[:len(*levels)-1]
	}

	rest := strings.TrimLeft(l, " \t")
	switch {
	case isThematicBreak(l):
		*levels = nil
	case listMarkerLen(rest) > 0 && orderedMarkerLen(rest) > 0:
		// At this indent with no list open the item would be indented code.
		if indent < indentedCodeWidth || len(*levels) > 0 {
			return r.renumber(l, indent, rest, levels)
		}
	case listMarkerLen(rest) > 0:
		// A bullet list takes over this level from an ordered one.
		for len(*levels) > 0 && (*levels)[len(*levels)-1].indent >= indent {
			*levels = (*levels)[:len(*levels)-1]
		}
	case afterBlank && indent == 0:
		*levels = nil // a new top-level block ends the list
	}
	return l
}

// renumber rewrites the ordered marker of l, opening a new list level when the
// line does not continue the level on top of the stack.
func (r orderedListNumbering) renumber(l string, indent int, rest string, levels *[]orderedLevel) string {
	markerLen := orderedMarkerLen(rest)
	delim := rest[markerLen-1]
	start, err := strconv.Atoi(rest[:markerLen-1])
	if err != nil {
		return l // more digits than an int holds; leave it alone
	}

	top := len(*levels) - 1
	if top < 0 || (*levels)[top].indent != indent || (*levels)[top].delim != delim {
		if top >= 0 && (*levels)[top].indent == indent {
			*levels = (*levels)[:top] // same level, different delimiter: new list
		}
		*levels = append(*levels, orderedLevel{indent: indent, delim: delim, num: start})
		top = len(*levels) - 1
	}

	lvl := &(*levels)[top]
	num := lvl.num
	lvl.num++
	return l[:len(l)-len(rest)] + strconv.Itoa(num) + string(delim) + rest[markerLen:]
}

// leadingIndent returns the indentation width of a line, counting a tab as four
// columns.
func leadingIndent(l string) int {
	n := 0
	for i := 0; i < len(l); i++ {
		switch l[i] {
		case ' ':
			n++
		case '\t':
			n += indentedCodeWidth
		default:
			return n
		}
	}
	return n
}
