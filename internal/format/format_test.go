package format

import (
	"strconv"
	"testing"

	"github.com/spf13/viper"
)

func defaultEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := Build(viper.New())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return e
}

func format(t *testing.T, src string) string {
	t.Helper()
	out, err := defaultEngine(t).Format([]byte(src))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	return string(out)
}

func TestEngine_EndToEnd(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "sentence per line",
			input: "One sentence. Two sentences. Three.\n",
			want:  "One sentence.\nTwo sentences.\nThree.\n",
		},
		{
			name:  "abbreviations and versions not split",
			input: "Use e.g. this and v1.2.3 now. Done.\n",
			want:  "Use e.g. this and v1.2.3 now.\nDone.\n",
		},
		{
			name:  "collapse blank lines to one",
			input: "a\n\n\n\nb\n",
			want:  "a\n\nb\n",
		},
		{
			name:  "blank line around heading",
			input: "text\n## Head\nmore\n",
			want:  "text\n\n## Head\n\nmore\n",
		},
		{
			name:  "unordered markers normalized",
			input: "* a\n+ b\n- c\n",
			want:  "- a\n- b\n- c\n",
		},
		{
			name:  "ordered list is renumbered consecutively",
			input: "1. first\n3. second\n5. third\n",
			want:  "1. first\n2. second\n3. third\n",
		},
		{
			name:  "ordered list keeps the number it starts at",
			input: "3. first\n9. second\n",
			want:  "3. first\n4. second\n",
		},
		{
			name:  "nested ordered lists are numbered independently",
			input: "1. outer\n   4. inner\n   9. inner two\n7. outer two\n",
			want:  "1. outer\n   4. inner\n   5. inner two\n2. outer two\n",
		},
		{
			name:  "a top-level paragraph ends the ordered list",
			input: "1. a\n7. b\n\npara\n\n5. c\n9. d\n",
			want:  "1. a\n2. b\n\npara\n\n5. c\n6. d\n",
		},
		{
			name:  "ordered markers inside a fence are untouched",
			input: "```\n1. one\n9. nine\n```\n",
			want:  "```\n1. one\n9. nine\n```\n",
		},
		{
			name:  "atx heading spacing and closing hashes",
			input: "##   Title  ##\n",
			want:  "## Title\n",
		},
		{
			name:  "setext headings become atx",
			input: "Title\n=====\n\nSubtitle\n--------\n",
			want:  "# Title\n\n## Subtitle\n",
		},
		{
			name:  "wrapped setext heading is joined",
			input: "A long title that\nwraps over lines\n===\n",
			want:  "# A long title that wraps over lines\n",
		},
		{
			name:  "thematic break after a blank line stays a break",
			input: "para\n\n---\n\nmore\n",
			want:  "para\n\n---\n\nmore\n",
		},
		{
			name:  "underline after a list item is not a heading",
			input: "- item\n---\n",
			want:  "- item\n---\n",
		},
		{
			name:  "setext underline inside a fence is untouched",
			input: "```\nTitle\n=====\n```\n",
			want:  "```\nTitle\n=====\n```\n",
		},
		{
			name:  "fenced code untouched",
			input: "```\nkeep. this.  as-is\n```\n",
			want:  "```\nkeep. this.  as-is\n```\n",
		},
		{
			name:  "hard tab expands to the next tab stop",
			input: "-\tItem with a tab\n",
			want:  "-   Item with a tab\n",
		},
		{
			name:  "leading tab expands to a full indent",
			input: "\tindented by a tab\n",
			want:  "    indented by a tab\n",
		},
		{
			name:  "tab inside an inline code span is kept",
			input: "Inline `a\tb` stays.\n",
			want:  "Inline `a\tb` stays.\n",
		},
		{
			name:  "tab inside a fence is kept",
			input: "```\ncol\tcol\n```\n",
			want:  "```\ncol\tcol\n```\n",
		},
		{
			name:  "inline code protects punctuation",
			input: "Call `a. b` then stop. Go.\n",
			want:  "Call `a. b` then stop.\nGo.\n",
		},
		{
			name:  "list continuation is indented",
			input: "- First. Second.\n",
			want:  "- First.\n  Second.\n",
		},
		{
			name:  "hard-wrapped paragraph is stitched",
			input: "Zero dependencies, works in browser and\nNode.js.\n",
			want:  "Zero dependencies, works in browser and Node.js.\n",
		},
		{
			name:  "wrapped blockquote is stitched",
			input: "> as easy as writing an email, by converting it\n> automatically into HTML.\n",
			want:  "> as easy as writing an email, by converting it automatically into HTML.\n",
		},
		{
			name:  "wrapped list item is stitched, items stay apart",
			input: "- item one that is\n  wrapped here\n- item two\n",
			want:  "- item one that is wrapped here\n- item two\n",
		},
		{
			name:  "hard line break is not stitched",
			input: "before the break  \nafter it\n",
			want:  "before the break  \nafter it\n",
		},
		{
			name:  "hard line break survives mid-paragraph stitching",
			input: "wrapped line one\nend of the break  \nafter it\n",
			want:  "wrapped line one end of the break  \nafter it\n",
		},
		{
			name:  "indented code block is left verbatim",
			input: "text\n\n    code. line one\n    code. line two\n",
			want:  "text\n\n    code. line one\n    code. line two\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := format(t, tt.input); got != tt.want {
				t.Errorf("\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestTableWidth(t *testing.T) {
	in := "| a | bb |\n|---|--:|\n| longer | 1 |\n"
	// Columns pad to widest content + 1 (default padding).
	want := "| a       |  bb |\n| ------- | --: |\n| longer  |   1 |\n"
	if got := format(t, in); got != want {
		t.Errorf("\n got: %q\nwant: %q", got, want)
	}
}

func TestBuild_UnknownRule(t *testing.T) {
	v := viper.New()
	v.Set("rules", []string{"does-not-exist"})
	if _, err := Build(v); err == nil {
		t.Fatal("expected error for unknown rule")
	}
}

func TestSemBr_ConfigurableBreakOn(t *testing.T) {
	v := viper.New()
	v.Set("rules", []string{"semantic-line-breaks"})
	v.Set("options.semantic-line-breaks.break-on", []string{"colon", "semicolon"})
	e, err := Build(v)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	out, err := e.Format([]byte("Setup: install; run. Keep.\n"))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	// Only colon and semicolon break; the sentence period does not.
	want := "Setup:\ninstall;\nrun. Keep.\n"
	if string(out) != want {
		t.Errorf("\n got: %q\nwant: %q", out, want)
	}
}

func TestOrderedListNumbering_Styles(t *testing.T) {
	const in = "3. first\n7. second\n8. third\n"
	tests := []struct {
		style string
		want  string
	}{
		{style: "increment", want: "3. first\n4. second\n5. third\n"},
		{style: "keep", want: in},
	}
	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			v := viper.New()
			v.Set("rules", []string{"ordered-list-numbering"})
			v.Set("options.ordered-list-numbering.style", tt.style)
			e, err := Build(v)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			out, err := e.Format([]byte(in))
			if err != nil {
				t.Fatalf("Format: %v", err)
			}
			if string(out) != tt.want {
				t.Errorf("\n got: %q\nwant: %q", out, tt.want)
			}
		})
	}
}

func TestHardTabs_Width(t *testing.T) {
	// A tab advances to the next tab stop, so the spaces it becomes depend on
	// the column it starts at, not only on the configured width.
	const in = "ab\tc\n"
	tests := []struct {
		width int
		want  string
	}{
		{width: 4, want: "ab  c\n"},
		{width: 2, want: "ab  c\n"},
		{width: 8, want: "ab      c\n"},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.width), func(t *testing.T) {
			v := viper.New()
			v.Set("rules", []string{"hard-tabs"})
			v.Set("options.hard-tabs.width", tt.width)
			e, err := Build(v)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			out, err := e.Format([]byte(in))
			if err != nil {
				t.Fatalf("Format: %v", err)
			}
			if string(out) != tt.want {
				t.Errorf("\n got: %q\nwant: %q", out, tt.want)
			}
		})
	}
}

func TestATXHeadings_TrailingPunctuation(t *testing.T) {
	tests := []struct {
		name  string
		punct any // nil = leave the option unset (default set)
		input string
		want  string
	}{
		{name: "default strips colon", input: "## Overview:\n", want: "## Overview\n"},
		{name: "default strips a run", input: "## Ship it!?\n", want: "## Ship it\n"},
		{name: "punctuation inside is kept", input: "## Ratio 1:2 rule\n", want: "## Ratio 1:2 rule\n"},
		{name: "escaped mark is kept", input: `## Done\.` + "\n", want: `## Done\.` + "\n"},
		{name: "closing hashes then punctuation", input: "## Overview: ##\n", want: "## Overview\n"},
		{name: "heading of only punctuation empties", input: "## ...\n", want: "##\n"},
		{name: "custom set", punct: ":", input: "## Keep it!\n", want: "## Keep it!\n"},
		{name: "custom set strips", punct: ":", input: "## Keep it:\n", want: "## Keep it\n"},
		{name: "empty set disables", punct: "", input: "## Overview:\n", want: "## Overview:\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.Set("rules", []string{"atx-headings"})
			if tt.punct != nil {
				v.Set("options.atx-headings.strip-trailing-punctuation", tt.punct)
			}
			e, err := Build(v)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			out, err := e.Format([]byte(tt.input))
			if err != nil {
				t.Fatalf("Format: %v", err)
			}
			if string(out) != tt.want {
				t.Errorf("\n got: %q\nwant: %q", out, tt.want)
			}
		})
	}
}
