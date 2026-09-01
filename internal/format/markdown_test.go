package format

import (
	"slices"
	"testing"
)

func TestSplitIndent(t *testing.T) {
	tests := []struct {
		line       string
		wantIndent string
		wantRest   string
	}{
		{"", "", ""},
		{"text", "", "text"},
		{"    text", "    ", "text"},
		{"  \t- item", "  \t", "- item"},
		{"   ", "   ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			indent, rest := splitIndent(tt.line)
			if indent != tt.wantIndent || rest != tt.wantRest {
				t.Errorf("splitIndent(%q) = %q, %q; want %q, %q",
					tt.line, indent, rest, tt.wantIndent, tt.wantRest)
			}
		})
	}
}

func TestIndentWidth(t *testing.T) {
	tests := []struct {
		indent string
		want   int
	}{
		{"", 0},
		{"   ", 3},
		{"\t", indentedCodeWidth},
		{" \t", 1 + indentedCodeWidth},
		{"\t\t", 2 * indentedCodeWidth},
	}
	for _, tt := range tests {
		t.Run(tt.indent, func(t *testing.T) {
			if got := indentWidth(tt.indent); got != tt.want {
				t.Errorf("indentWidth(%q) = %d, want %d", tt.indent, got, tt.want)
			}
		})
	}
}

func TestLeadingIndent(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{"text", 0},
		{"  text", 2},
		{"\t1. item", indentedCodeWidth},
		{"", 0},
		{"   ", 3}, // a whitespace-only line is all indentation
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := leadingIndent(tt.line); got != tt.want {
				t.Errorf("leadingIndent(%q) = %d, want %d", tt.line, got, tt.want)
			}
		})
	}
}

func TestScanPrefix(t *testing.T) {
	tests := []struct {
		line                              string
		wantQuoteEnd, wantIndent, wantEnd int
	}{
		{"text", 0, 0, 0},
		{"    code", 0, indentedCodeWidth, 4},
		{"\ttab", 0, indentedCodeWidth, 1},
		{"> quoted", 1, 1, 2},
		{"> >  nested", 3, 2, 5},
		{">", 1, 0, 1},          // a marker with nothing after it
		{"  > quoted", 3, 1, 4}, // space before a marker is not indent
		{">     code", 1, 5, 6}, // indented code inside a quote
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			quoteEnd, indent, contentStart := scanPrefix(tt.line)
			if quoteEnd != tt.wantQuoteEnd || indent != tt.wantIndent || contentStart != tt.wantEnd {
				t.Errorf("scanPrefix(%q) = %d, %d, %d; want %d, %d, %d", tt.line,
					quoteEnd, indent, contentStart, tt.wantQuoteEnd, tt.wantIndent, tt.wantEnd)
			}
		})
	}
}

func TestListMarkerLen(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"", 0},
		{"- item", 2},
		{"*  item", 3},
		{"+\titem", 2},
		{"1. item", 3},
		{"12)  item", 5},
		{"-item", 0}, // no whitespace after the marker
		{"-", 0},
		{"1.", 0},
		{"prose", 0},
		{"- ", 2}, // an empty item is still a marker
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := listMarkerLen(tt.s); got != tt.want {
				t.Errorf("listMarkerLen(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestOrderedMarkerLen(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"", 0},
		{"1. item", 2},
		{"42. item", 3},
		{"7) item", 2},
		{"1", 0},   // no delimiter
		{"1a.", 0}, // delimiter must follow the digits directly
		{".", 0},
		{"- item", 0},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := orderedMarkerLen(tt.s); got != tt.want {
				t.Errorf("orderedMarkerLen(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestIsThematicBreak(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"---", true},
		{"***", true},
		{"___", true},
		{"- - -", true},
		{"  ***  ", true},
		{"----------", true},
		{"--", false},  // too short
		{"-*-", false}, // mixed characters
		{"--- text", false},
		{"", false},
		{"- item", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isThematicBreak(tt.line); got != tt.want {
				t.Errorf("isThematicBreak(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestFenceMarker(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"```", "```"},
		{"```go", "```"},
		{"````", "````"},
		{"~~~", "~~~"},
		{"~~~~yaml", "~~~~"},
		{"``", ""}, // shorter than three: an inline code span
		{"~~", ""},
		{"text", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := fenceMarker(tt.s); got != tt.want {
				t.Errorf("fenceMarker(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseATXHeading(t *testing.T) {
	tests := []struct {
		line       string
		wantHashes string
		wantText   string
		wantOK     bool
	}{
		{"# Title", "#", "Title", true},
		{"###   Spaced   ", "###", "Spaced", true},
		{"## Title ##", "##", "Title", true},
		{"###### Six", "######", "Six", true},
		{"   # Indented three", "#", "Indented three", true},
		{"# C#", "#", "C#", true}, // closing run needs a space before it
		{"# ###", "#", "", true},  // text that is only hashes
		{"#", "#", "", true},      // a bare hash is an empty heading
		{"#Title", "", "", false}, // CommonMark needs a space
		{"####### 7", "", "", false},
		{"    # Code", "", "", false},
		{"text", "", "", false},
		{"", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			hashes, text, ok := parseATXHeading(tt.line)
			if hashes != tt.wantHashes || text != tt.wantText || ok != tt.wantOK {
				t.Errorf("parseATXHeading(%q) = %q, %q, %v; want %q, %q, %v",
					tt.line, hashes, text, ok, tt.wantHashes, tt.wantText, tt.wantOK)
			}
		})
	}
}

func TestIsIndented(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"    code", true},
		{"\tcode", true},
		{">     quoted code", true},
		{"   three spaces", false},
		{"    - item", false}, // a marker makes it nesting, not code
		{"text", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isIndented(tt.line); got != tt.want {
				t.Errorf("isIndented(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestInlineProtected(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string // "." unprotected, "x" protected, aligned under s
	}{
		{"code span", "a `b. c` d", "..xxxxxx.."},
		{"interior run does not close the span", "`a```b`", "xxxxxxx"},
		{"double backticks", "a ``b`c`` d", "..xxxxxxx.."},
		{"unclosed backtick", "a `b c", "......"},
		{"autolink", "see <http://x> ok", "....xxxxxxxxxx..."},
		{"angle with space is not an autolink", "a <b c> d", "........."},
		{"link destination", "[a](b. c) d", "...xxxxxx.."},
		{"plain prose", "nothing here", "............"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inlineProtected(tt.s)
			if len(got) != len(tt.want) {
				t.Fatalf("inlineProtected(%q) has length %d, want %d", tt.s, len(got), len(tt.want))
			}
			for i, p := range got {
				if p != (tt.want[i] == 'x') {
					t.Errorf("inlineProtected(%q)[%d] = %v (byte %q), want %v",
						tt.s, i, p, tt.s[i], tt.want[i] == 'x')
				}
			}
		})
	}
}

func TestRunLen(t *testing.T) {
	tests := []struct {
		name string
		s    string
		i    int
		ch   byte
		want int
	}{
		{"run at start", "~~~~go", 0, '~', 4},
		{"run in the middle", "a```b", 1, '`', 3},
		{"no run", "a```b", 0, '`', 0},
		{"past the end", "abc", 3, 'c', 0},
		{"to the end", "ab", 1, 'b', 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runLen(tt.s, tt.i, tt.ch); got != tt.want {
				t.Errorf("runLen(%q, %d, %q) = %d, want %d", tt.s, tt.i, tt.ch, got, tt.want)
			}
		})
	}
}

func TestFindRun(t *testing.T) {
	tests := []struct {
		name string
		s    string
		from int
		ch   byte
		n    int
		want int
	}{
		{"exact run", "a`b", 0, '`', 1, 1},
		{"skips a longer run", "a```b`c", 0, '`', 1, 5}, // not the tail of the run of three
		{"skips a run that starts before from", "a``b", 2, '`', 1, -1},
		{"finds a pair", "a``b", 0, '`', 2, 1},
		{"honors from", "`a`b", 1, '`', 1, 2},
		{"missing", "abc", 0, '`', 1, -1},
		{"wrong length only", "a``b", 0, '`', 3, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findRun(tt.s, tt.from, tt.ch, tt.n); got != tt.want {
				t.Errorf("findRun(%q, %d, %q, %d) = %d, want %d",
					tt.s, tt.from, tt.ch, tt.n, got, tt.want)
			}
		})
	}
}

func TestMarkRange(t *testing.T) {
	tests := []struct {
		name   string
		length int
		lo, hi int
		want   []bool
	}{
		{"inside", 4, 1, 3, []bool{false, true, true, false}},
		{"clamped to the end", 3, 1, 9, []bool{false, true, true}},
		{"empty range", 3, 2, 2, []bool{false, false, false}},
		{"whole slice", 2, 0, 2, []bool{true, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := make([]bool, tt.length)
			markRange(p, tt.lo, tt.hi)
			if !slices.Equal(p, tt.want) {
				t.Errorf("markRange(len %d, %d, %d) = %v, want %v",
					tt.length, tt.lo, tt.hi, p, tt.want)
			}
		})
	}
}
