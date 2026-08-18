package format

import (
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
			name:  "atx heading spacing and closing hashes",
			input: "##   Title  ##\n",
			want:  "## Title\n",
		},
		{
			name:  "fenced code untouched",
			input: "```\nkeep. this.  as-is\n```\n",
			want:  "```\nkeep. this.  as-is\n```\n",
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
