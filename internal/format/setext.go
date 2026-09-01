package format

import (
	"strings"

	"github.com/spf13/viper"
)

const nameSetextHeadings = "setext-headings"

func init() { Register(nameSetextHeadings, newSetextHeadings) }

// setextHeadings rewrites Setext (underlined) headings as ATX headings so all
// headings in a document use one style (markdownlint MD003):
//
//	Title      ->  # Title
//	=====
//
//	Subtitle   ->  ## Subtitle
//	--------
//
// A Setext heading's text is the whole paragraph preceding the underline, so a
// hard-wrapped one is joined into the single line an ATX heading requires.
// Underlines inside verbatim spans, and underlines that follow anything other
// than a paragraph (a list item, a blockquote, a table row, front matter), are
// left alone — there the "---" is a thematic break or block content, not a
// heading.
type setextHeadings struct{}

func newSetextHeadings(_ *viper.Viper) (Rule, error) { return setextHeadings{}, nil }

func (setextHeadings) Name() string { return nameSetextHeadings }

func (setextHeadings) Apply(lines []string) ([]string, error) {
	mask := codeMask(lines)
	out := make([]string, 0, len(lines))
	var para []string // the paragraph an underline would turn into a heading

	flush := func() {
		out = append(out, para...)
		para = para[:0]
	}
	for i, l := range lines {
		switch {
		case mask[i]:
			flush()
			out = append(out, l)
		case len(para) > 0 && setextLevel(l) > 0:
			out = append(out, strings.Repeat("#", setextLevel(l))+" "+joinParagraph(para))
			para = para[:0]
		case isParagraphLine(l):
			para = append(para, l)
		default:
			flush()
			out = append(out, l)
		}
	}
	flush()
	return out, nil
}

// setextLevel returns the heading level of a Setext underline — 1 for a run of
// "=", 2 for a run of "-" — or 0 when l is not an underline. A "-" run is also a
// thematic break; the caller decides which it is by whether a paragraph precedes
// it, as CommonMark does.
func setextLevel(l string) int {
	indent, s := splitIndent(strings.TrimRight(l, " \t"))
	if indentWidth(indent) > 3 { // indented far enough to be code, not an underline
		return 0
	}
	if s == "" {
		return 0
	}
	for i := 0; i < len(s); i++ {
		if s[i] != s[0] {
			return 0
		}
	}
	switch s[0] {
	case '=':
		return 1
	case '-':
		return 2
	}
	return 0
}

// isParagraphLine reports whether l can be part of the paragraph that a Setext
// underline turns into a heading. Anything that opens a different block cannot,
// and neither can quoted content: the underline below it sits outside the quote,
// where it is a thematic break rather than that paragraph's underline.
func isParagraphLine(l string) bool {
	return classify(l) == kindParagraph && quoteDepth(l) == 0
}

// joinParagraph collapses a hard-wrapped paragraph into the single line an ATX
// heading needs.
func joinParagraph(para []string) string {
	parts := make([]string, len(para))
	for i, l := range para {
		parts[i] = strings.TrimSpace(l)
	}
	return strings.Join(parts, " ")
}
