package testcases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/drackthor/mdformat/internal/format"
	"github.com/spf13/viper"
)

// fixture is one case from test-config.yaml: formatting inputs/Input under
// Config must produce expected/Expected. Config is optional and takes the same
// shape as .mdformat.yaml.
type fixture struct {
	Input    string
	Expected string
	Config   map[string]any
}

// TestFixtures runs every case listed in test-config.yaml. The expected files
// are written by hand; never generate them by running the formatter.
func TestFixtures(t *testing.T) {
	cfg := viper.New()
	cfg.SetConfigFile("test-config.yaml")
	if err := cfg.ReadInConfig(); err != nil {
		t.Fatalf("read test-config.yaml: %v", err)
	}
	var fixtures []fixture
	if err := cfg.UnmarshalKey("cases", &fixtures); err != nil {
		t.Fatalf("parse cases: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("test-config.yaml lists no cases")
	}

	for _, f := range fixtures {
		t.Run(f.Expected, func(t *testing.T) {
			engine, err := f.engine()
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			src, err := os.ReadFile(filepath.Join("inputs", f.Input))
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join("expected", f.Expected))
			if err != nil {
				t.Fatal(err)
			}

			got, err := engine.Format(src)
			if err != nil {
				t.Fatalf("Format: %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("output mismatch\n got: %q\nwant: %q", got, want)
			}

			// Formatting must be idempotent.
			twice, err := engine.Format(got)
			if err != nil {
				t.Fatalf("Format (2nd pass): %v", err)
			}
			if string(twice) != string(got) {
				t.Errorf("not idempotent\n first: %q\nsecond: %q", got, twice)
			}
		})
	}
}

func (f fixture) engine() (*format.Engine, error) {
	v := viper.New()
	if len(f.Config) > 0 {
		if err := v.MergeConfigMap(f.Config); err != nil {
			return nil, err
		}
	}
	return format.Build(v)
}
