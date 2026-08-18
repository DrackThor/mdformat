package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ProjectOverridesUser(t *testing.T) {
	dir := t.TempDir()
	xdg := filepath.Join(dir, "xdg")
	if err := os.MkdirAll(filepath.Join(xdg, "mdformat"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)

	// User config sets padding 5; project config overrides padding to 2.
	writeFile(t, filepath.Join(xdg, "mdformat", "config.yaml"),
		"options:\n  table-width:\n    padding: 5\n  list-markers:\n    marker: \"*\"\n")

	cwd := t.TempDir()
	chdir(t, cwd)
	writeFile(t, ".mdformat.yaml", "options:\n  table-width:\n    padding: 2\n")

	v, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := v.GetInt("options.table-width.padding"); got != 2 {
		t.Errorf("padding = %d, want 2 (project overrides user)", got)
	}
	if got := v.GetString("options.list-markers.marker"); got != "*" {
		t.Errorf("marker = %q, want %q (user value kept when project omits it)", got, "*")
	}
	if len(v.GetStringSlice("rules")) == 0 {
		t.Error("expected default rules to be applied")
	}
}

func TestLoad_NoConfigUsesDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	chdir(t, t.TempDir())
	v, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(v.GetStringSlice("rules")) == 0 {
		t.Error("expected default rules with no config files present")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}
