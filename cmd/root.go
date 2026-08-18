// Package cmd wires the mdformat command-line interface using Cobra and Viper.
package cmd

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
	RunE: func(_ *cobra.Command, args []string) error {
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

		files, err := collectFiles(args)
		if err != nil {
			return err
		}

		changed := 0
		for _, f := range files {
			wasChanged, err := formatFile(engine, f, checkOnly)
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
// set. It returns whether the file's content changed.
func formatFile(engine *format.Engine, path string, checkOnly bool) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	out, err := engine.Format(src)
	if err != nil {
		return false, fmt.Errorf("format %s: %w", path, err)
	}
	if bytes.Equal(out, src) {
		return false, nil
	}
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
