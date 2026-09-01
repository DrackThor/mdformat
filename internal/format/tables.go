package format

import (
	"strings"
	"unicode/utf8"

	"github.com/spf13/viper"
)

const nameTableWidth = "table-width"

func init() { Register(nameTableWidth, newTableWidth) }

type alignment int

const (
	alignLeft alignment = iota
	alignRight
	alignCenter
)

// tableWidth normalizes GFM pipe tables so every cell in a column is padded with
// spaces to the width of that column's widest cell plus a configurable padding
// (default 1). Column alignment markers in the delimiter row are preserved.
//
// ponytail: cell width is counted in runes, so full-width CJK/emoji columns may
// look under-padded. Upgrade to a display-width measure if that matters.
type tableWidth struct {
	padding int
}

func newTableWidth(opts *viper.Viper) (Rule, error) {
	p := 1
	if opts != nil && opts.IsSet("padding") {
		p = opts.GetInt("padding")
		if p < 0 {
			p = 0
		}
	}
	return tableWidth{padding: p}, nil
}

func (tableWidth) Name() string { return nameTableWidth }

func (r tableWidth) Apply(lines []string) ([]string, error) {
	mask := codeMask(lines)
	out := make([]string, 0, len(lines))
	i := 0
	for i < len(lines) {
		if !mask[i] && i+1 < len(lines) && !mask[i+1] &&
			looksLikeRow(lines[i]) && isDelimiterRow(lines[i+1]) {
			end := i + 2
			for end < len(lines) && !mask[end] && looksLikeRow(lines[end]) {
				end++
			}
			out = append(out, r.formatTable(lines[i:end])...)
			i = end
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return out, nil
}

func (r tableWidth) formatTable(block []string) []string {
	indent, _ := splitIndent(block[0])

	header := splitRow(block[0])
	aligns := parseAligns(block[1])
	ncols := len(header)
	if len(aligns) > ncols {
		ncols = len(aligns)
	}

	// Body rows are every row except the delimiter (index 1).
	rows := [][]string{header}
	for _, l := range block[2:] {
		rows = append(rows, splitRow(l))
	}

	widths := make([]int, ncols)
	for _, row := range rows {
		for c := 0; c < ncols; c++ {
			if c < len(row) {
				if w := utf8.RuneCountInString(row[c]); w > widths[c] {
					widths[c] = w
				}
			}
		}
	}
	for c := range widths {
		widths[c] += r.padding
		if widths[c] < 3 { // keep delimiter cells valid (":-:" needs 3)
			widths[c] = 3
		}
	}

	out := make([]string, 0, len(block))
	out = append(out,
		indent+renderRow(header, widths, aligns),
		indent+renderDelimiter(widths, aligns),
	)
	for _, row := range rows[1:] {
		out = append(out, indent+renderRow(row, widths, aligns))
	}
	return out
}

func renderRow(cells []string, widths []int, aligns []alignment) string {
	var b strings.Builder
	b.WriteByte('|')
	for c := range widths {
		cell := ""
		if c < len(cells) {
			cell = cells[c]
		}
		b.WriteByte(' ')
		b.WriteString(pad(cell, widths[c], alignOf(aligns, c)))
		b.WriteString(" |")
	}
	return b.String()
}

func renderDelimiter(widths []int, aligns []alignment) string {
	var b strings.Builder
	b.WriteByte('|')
	for c := range widths {
		b.WriteByte(' ')
		w := widths[c]
		switch alignOf(aligns, c) {
		case alignLeft:
			b.WriteString(strings.Repeat("-", w))
		case alignRight:
			b.WriteString(strings.Repeat("-", w-1) + ":")
		case alignCenter:
			b.WriteString(":" + strings.Repeat("-", w-2) + ":")
		}
		b.WriteString(" |")
	}
	return b.String()
}

func alignOf(aligns []alignment, c int) alignment {
	if c < len(aligns) {
		return aligns[c]
	}
	return alignLeft
}

func pad(s string, width int, a alignment) string {
	gap := width - utf8.RuneCountInString(s)
	if gap <= 0 {
		return s
	}
	switch a {
	case alignRight:
		return strings.Repeat(" ", gap) + s
	case alignCenter:
		left := gap / 2
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", gap-left)
	default:
		return s + strings.Repeat(" ", gap)
	}
}

// looksLikeRow reports whether l is a candidate table row: non-blank and
// containing at least one unescaped pipe.
func looksLikeRow(l string) bool {
	t := strings.TrimSpace(l)
	if t == "" {
		return false
	}
	return len(splitRow(l)) > 0 && strings.ContainsRune(t, '|')
}

// isDelimiterRow reports whether l is a table delimiter row (each cell is
// optional colons around one or more dashes).
func isDelimiterRow(l string) bool {
	if !strings.ContainsRune(l, '-') {
		return false
	}
	cells := splitRow(l)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		c = strings.TrimSpace(c)
		c = strings.TrimPrefix(c, ":")
		c = strings.TrimSuffix(c, ":")
		if c == "" || strings.Trim(c, "-") != "" {
			return false
		}
	}
	return true
}

func parseAligns(l string) []alignment {
	cells := splitRow(l)
	aligns := make([]alignment, len(cells))
	for i, c := range cells {
		c = strings.TrimSpace(c)
		left := strings.HasPrefix(c, ":")
		right := strings.HasSuffix(c, ":")
		switch {
		case left && right:
			aligns[i] = alignCenter
		case right:
			aligns[i] = alignRight
		default:
			aligns[i] = alignLeft
		}
	}
	return aligns
}

// splitRow splits a table row into trimmed cell contents, honoring one optional
// leading and trailing pipe and backslash-escaped pipes within cells.
func splitRow(l string) []string {
	s := strings.TrimSpace(l)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")

	var cells []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '|' {
			cur.WriteString("\\|")
			i++
			continue
		}
		if s[i] == '|' {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(s[i])
	}
	cells = append(cells, strings.TrimSpace(cur.String()))
	return cells
}
