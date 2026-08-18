//go:build ignore

// Command gen_expected regenerates the golden files in test-cases/expected by
// running the default formatter over every file in test-cases/inputs.
//
// Usage:
//
//	go run ./scripts/gen_expected.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/drackthor/mdformat/internal/format"
	"github.com/spf13/viper"
)

func main() {
	root := "test-cases"
	inputs := filepath.Join(root, "inputs")
	expected := filepath.Join(root, "expected")

	engine, err := format.Build(viper.New())
	if err != nil {
		fatal(err)
	}

	entries, err := os.ReadDir(inputs)
	if err != nil {
		fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src, err := os.ReadFile(filepath.Join(inputs, e.Name()))
		if err != nil {
			fatal(err)
		}
		out, err := engine.Format(src)
		if err != nil {
			fatal(err)
		}
		if err := os.WriteFile(filepath.Join(expected, e.Name()), out, 0644); err != nil {
			fatal(err)
		}
		fmt.Printf("wrote %s\n", filepath.Join(expected, e.Name()))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
