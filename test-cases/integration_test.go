package testcases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/drackthor/mdformat/internal/format"
	"github.com/spf13/viper"
)

// TestFixtures formats every file in inputs/ with the default rule set and
// compares against the golden file of the same name in expected/. Regenerate
// the golden files with: go run ./scripts/gen_expected.go
func TestFixtures(t *testing.T) {
	engine, err := format.Build(viper.New())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	entries, err := os.ReadDir("inputs")
	if err != nil {
		t.Fatalf("read inputs: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("inputs", name))
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join("expected", name))
			if err != nil {
				t.Fatalf("missing golden file (run scripts/gen_expected.go): %v", err)
			}
			got, err := engine.Format(src)
			if err != nil {
				t.Fatalf("Format: %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("output mismatch for %s\n got: %q\nwant: %q", name, got, want)
			}

			// Formatting must be idempotent.
			twice, err := engine.Format(got)
			if err != nil {
				t.Fatalf("Format (2nd pass): %v", err)
			}
			if string(twice) != string(got) {
				t.Errorf("not idempotent for %s\n first: %q\nsecond: %q", name, got, twice)
			}
		})
	}
}
