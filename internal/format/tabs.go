package format

import (
	"strings"

	"github.com/spf13/viper"
)

const nameHardTabs = "hard-tabs"

func init() { Register(nameHardTabs, newHardTabs) }

// hardTabs replaces hard tabs with spaces (markdownlint MD010). A tab advances
// to the next tab stop rather than expanding to a fixed number of spaces, which
// is how Markdown itself reads tabs, so a tab keeps the column it lands on:
//
//	-\tItem  ->  -   Item
//
// Tabs inside verbatim spans are left alone: fenced code and front matter via
// [codeMask], and inline code spans, autolinks, and link destinations via
// [inlineProtected]. Their width still counts toward the tab stops that follow.
type hardTabs struct {
	width int
}

func newHardTabs(opts *viper.Viper) (Rule, error) {
	r := hardTabs{width: indentedCodeWidth}
	if opts != nil && opts.IsSet("width") {
		if w := opts.GetInt("width"); w >= 1 {
			r.width = w
		}
	}
	return r, nil
}

func (hardTabs) Name() string { return nameHardTabs }

func (r hardTabs) Apply(lines []string) ([]string, error) {
	mask := codeMask(lines)
	out := make([]string, len(lines))
	for i, l := range lines {
		if mask[i] || !strings.ContainsRune(l, '\t') {
			out[i] = l
			continue
		}
		out[i] = r.expand(l)
	}
	return out, nil
}

// expand rewrites the tabs in a line as the spaces that reach the next tab stop,
// leaving the ones inside inline verbatim spans as tabs.
func (r hardTabs) expand(l string) string {
	prot := inlineProtected(l)
	var b strings.Builder
	b.Grow(len(l))

	col := 0
	for i, ch := range l {
		if ch != '\t' {
			b.WriteRune(ch)
			col++
			continue
		}
		stop := r.width - col%r.width
		if prot[i] {
			b.WriteRune('\t')
		} else {
			b.WriteString(strings.Repeat(" ", stop))
		}
		col += stop
	}
	return b.String()
}
