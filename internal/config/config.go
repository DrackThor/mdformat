// Package config loads mdformat configuration with layered precedence using
// Viper. Lower layers provide defaults that upper layers override:
//
//  1. Built-in defaults.
//  2. User config: $XDG_CONFIG_HOME/mdformat/config.yaml
//     (falling back to ~/.config/mdformat/config.yaml).
//  3. Project config in the current directory: ./.mdformat.yaml
//  4. An explicit --config file.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/drackthor/mdformat/internal/format"
	"github.com/spf13/viper"
)

// Load builds a Viper instance with defaults applied and any discovered config
// files merged in precedence order. explicit is the value of --config; it may be
// empty. It returns an error only when a config file exists but cannot be read
// or parsed.
func Load(explicit string) (*viper.Viper, error) {
	v := viper.New()
	v.SetDefault("rules", format.DefaultRuleOrder)

	sources := []string{userConfigPath(), projectConfigPath(), explicit}
	loadedAny := false
	for _, path := range sources {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		v.SetConfigFile(path)
		if !loadedAny {
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("read config %s: %w", path, err)
			}
			loadedAny = true
			continue
		}
		if err := v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("merge config %s: %w", path, err)
		}
	}
	return v, nil
}

func projectConfigPath() string {
	return ".mdformat.yaml"
}

func userConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "mdformat", "config.yaml")
}
