package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/drackthor/mdformat/internal/format"
	"github.com/spf13/viper"
)

func TestLocated(t *testing.T) {
	many := make([]int, maxReportedLines+2)
	for i := range many {
		many[i] = i + 1
	}

	tests := []struct {
		name   string
		change format.RuleChange
		want   string
	}{
		{
			name:   "single line",
			change: format.RuleChange{Lines: []int{4}},
			want:   "line 4",
		},
		{
			name:   "several lines",
			change: format.RuleChange{Lines: []int{1, 3, 8}},
			want:   "lines 1, 3, 8",
		},
		{
			name:   "collapsed range",
			change: format.RuleChange{From: 3, To: 3},
			want:   "line 3",
		},
		{
			name:   "range",
			change: format.RuleChange{From: 4, To: 9},
			want:   "lines 4-9",
		},
		{
			name:   "capped list",
			change: format.RuleChange{Lines: many},
			want:   "lines 1, 2, 3, 4, 5, 6, 7, 8, 9, 10 (+2 more)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := located(tt.change); got != tt.want {
				t.Errorf("located(%+v) = %q, want %q", tt.change, got, tt.want)
			}
		})
	}
}

func TestPrintTrace(t *testing.T) {
	trace := format.Trace{
		{Rule: "trailing-whitespace", Lines: []int{1, 3}},
		{Rule: "blank-lines", From: 2, To: 4},
	}

	tests := []struct {
		name      string
		trace     format.Trace
		verbosity int
		want      string
	}{
		{
			name:      "silent by default",
			trace:     trace,
			verbosity: 0,
		},
		{
			name:      "empty trace prints nothing",
			trace:     nil,
			verbosity: 2,
		},
		{
			name:      "rule names",
			trace:     trace,
			verbosity: 1,
			want:      "doc.md: trailing-whitespace, blank-lines\n",
		},
		{
			name:      "one line per rule",
			trace:     trace,
			verbosity: 2,
			want: "doc.md  trailing-whitespace      lines 1, 3\n" +
				"doc.md  blank-lines              lines 2-4\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := printTrace("doc.md", tt.trace, tt.verbosity); got != tt.want {
				t.Errorf("printTrace = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatFile(t *testing.T) {
	engine, err := format.Build(viper.New())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	tests := []struct {
		name        string
		src         string
		checkOnly   bool
		wantChanged bool
		wantOnDisk  string
	}{
		{
			name:        "rewrites a file that needs formatting",
			src:         "Trailing space. \n",
			wantChanged: true,
			wantOnDisk:  "Trailing space.\n",
		},
		{
			name:        "leaves a formatted file alone",
			src:         "Already formatted.\n",
			wantChanged: false,
			wantOnDisk:  "Already formatted.\n",
		},
		{
			name:        "check mode reports without writing",
			src:         "Trailing space. \n",
			checkOnly:   true,
			wantChanged: true,
			wantOnDisk:  "Trailing space. \n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "doc.md")
			if err := os.WriteFile(path, []byte(tt.src), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			changed, err := formatFile(engine, path, tt.checkOnly, 0)
			if err != nil {
				t.Fatalf("formatFile: %v", err)
			}
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back: %v", err)
			}
			if string(got) != tt.wantOnDisk {
				t.Errorf("file = %q, want %q", got, tt.wantOnDisk)
			}
		})
	}
}

func TestFormatFile_MissingFile(t *testing.T) {
	engine, err := format.Build(viper.New())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := formatFile(engine, filepath.Join(t.TempDir(), "absent.md"), false, 0); err == nil {
		t.Fatal("formatFile on a missing file returned no error")
	}
}
