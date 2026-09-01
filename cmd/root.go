// Package cmd wires the mdformat command-line interface using Cobra and Viper.
package cmd

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/drackthor/mdformat/internal/config"
	"github.com/drackthor/mdformat/internal/format"
	"github.com/drackthor/mdformat/internal/version"
	"github.com/spf13/cobra"
)

var (
	configPath  string
	recursive   bool
	checkOnly   bool
	showVersion bool
	verbosity   int
)

var markdownExts = map[string]bool{".md": true, ".markdown": true}

var rootCmd = &cobra.Command{
	Use:           "mdformat [flags] <file|dir>...",
	Short:         "Format Markdown files in place using configurable, ordered rules",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `mdformat formats Markdown files in place.

Rules (semantic line breaks, table width, blank-line normalization, and more)
are applied in a configured order and can be enabled or disabled individually
via a config file. Configuration is read, in increasing precedence, from
built-in defaults, $XDG_CONFIG_HOME/mdformat/config.yaml, ./.mdformat.yaml, and
an explicit --config file.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			return cobra.NoArgs(cmd, args)
		}
		return cobra.MinimumNArgs(1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.String())
			return nil
		}

		v, err := config.Load(configPath)
		if err != nil {
			return err
		}
		engine, err := format.Build(v)
		if err != nil {
			return err
		}
		// The flag wins over a "verbosity" key in the config file.
		if !cmd.Flags().Changed("verbose") {
			verbosity = v.GetInt("verbosity")
		}

		files, err := collectFiles(args)
		if err != nil {
			return err
		}

		changed := 0
		for _, f := range files {
			wasChanged, err := formatFile(engine, f, checkOnly, verbosity)
			if err != nil {
				return err
			}
			if wasChanged {
				changed++
			}
		}

		if checkOnly && changed > 0 {
			return fmt.Errorf("%d file(s) would be reformatted", changed)
		}
		return nil
	},
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "explicit config file (highest precedence)")
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "recurse into directories")
	rootCmd.Flags().BoolVar(&checkOnly, "check", false, "report files needing formatting without writing; exit non-zero if any")
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "print mdformat version and exit")
	rootCmd.Flags().CountVarP(&verbosity, "verbose", "v",
		"name the rules that changed each file; repeat (-vv) to add the lines they changed")
	rootCmd.AddCommand(newVersionCommand())
}

// collectFiles expands the given paths into a deduplicated list of Markdown
// files. Directories are scanned for *.md/*.markdown (recursively with -r).
func collectFiles(paths []string) ([]string, error) {
	seen := map[string]bool{}
	var files []string
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", p, err)
		}
		if !info.IsDir() {
			add(p)
			continue
		}
		if err := walkDir(p, add); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func walkDir(dir string, add func(string)) error {
	if !recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("read dir %s: %w", dir, err)
		}
		for _, e := range entries {
			if !e.IsDir() && isMarkdown(e.Name()) {
				add(filepath.Join(dir, e.Name()))
			}
		}
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isMarkdown(d.Name()) {
			add(path)
		}
		return nil
	})
}

func isMarkdown(name string) bool {
	return markdownExts[strings.ToLower(filepath.Ext(name))]
}

// formatFile formats a single file, writing changes in place unless checkOnly is
// set. It returns whether the file's content changed. What the rules did is
// reported on stderr from verbosity 1 upward, leaving stdout to the file lines a
// pipeline consumes.
func formatFile(engine *format.Engine, path string, checkOnly bool, verbosity int) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	out, trace, err := engine.FormatTrace(src)
	if err != nil {
		return false, fmt.Errorf("format %s: %w", path, err)
	}
	if bytes.Equal(out, src) {
		return false, nil
	}
	fmt.Fprint(os.Stderr, printTrace(path, trace, verbosity))
	if checkOnly {
		fmt.Printf("would reformat %s\n", path)
		return true, nil
	}
	info, statErr := os.Stat(path)
	mode := fs.FileMode(0644)
	if statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(path, out, mode); err != nil {
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("formatted %s\n", path)
	return true, nil
}

// printTrace reports what each rule changed: the rule names at verbosity 1, one
// line per rule with the lines it touched above that. It returns "" below
// verbosity 1 and for a file no rule changed.
//
// For example, a file two rules changed renders
// "doc.md: trailing-whitespace, blank-lines\n" at verbosity 1, and at
// verbosity 2 one line per rule ending in the [located] description of its
// changes.
//
// See TestPrintTrace.
func printTrace(path string, trace format.Trace, verbosity int) string {
	if verbosity < 1 || len(trace) == 0 {
		return ""
	}
	if verbosity == 1 {
		names := make([]string, len(trace))
		for i, change := range trace {
			names[i] = change.Rule
		}
		return fmt.Sprintf("%s: %s\n", path, strings.Join(names, ", "))
	}

	var b strings.Builder
	for _, change := range trace {
		fmt.Fprintf(&b, "%s  %-24s %s\n", path, change.Rule, located(change))
	}
	return b.String()
}

// maxReportedLines caps how many line numbers one rule reports, so a rule that
// touches every line of a long document stays readable.
const maxReportedLines = 10

// located renders where a rule made its changes.
//
// For example, a rule that rewrote lines 1 and 3 in place renders
// "lines 1, 3", a rule that changed the line count renders its range as
// "lines 4-9", and anything past [maxReportedLines] line numbers is summarized
// as a trailing " (+N more)".
//
// See TestLocated.
func located(change format.RuleChange) string {
	if len(change.Lines) == 0 {
		if change.From == change.To {
			return fmt.Sprintf("line %d", change.From)
		}
		return fmt.Sprintf("lines %d-%d", change.From, change.To)
	}

	shown := change.Lines
	suffix := ""
	if len(shown) > maxReportedLines {
		suffix = fmt.Sprintf(" (+%d more)", len(shown)-maxReportedLines)
		shown = shown[:maxReportedLines]
	}
	numbers := make([]string, len(shown))
	for i, n := range shown {
		numbers[i] = strconv.Itoa(n)
	}
	label := "lines"
	if len(change.Lines) == 1 {
		label = "line"
	}
	return label + " " + strings.Join(numbers, ", ") + suffix
}
