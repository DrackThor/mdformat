// Package format applies an ordered, configurable set of Markdown formatting
// rules. Each rule implements [Rule] and is registered by name so the set can
// be expanded without touching the engine or the CLI.
package format

import (
	"fmt"

	"github.com/spf13/viper"
)

// Rule is a single, independent Markdown transformation. Rules operate on the
// document as a slice of lines (without trailing newlines) and return the
// transformed lines. A rule must preserve the rendered meaning of the document
// and leave verbatim spans (fenced code, front matter) untouched.
type Rule interface {
	// Name returns the rule's kebab-case identifier, e.g. "table-width".
	Name() string
	// Apply transforms the document lines.
	Apply(lines []string) ([]string, error)
}

// Factory builds a Rule from its options sub-tree (may be nil when the config
// omits an options block for the rule; constructors must then use defaults).
type Factory func(opts *viper.Viper) (Rule, error)

var registry = map[string]Factory{}

// Register makes a rule constructor available to [Build] under name. It is
// called from rule files' init functions.
func Register(name string, f Factory) {
	registry[name] = f
}

// DefaultRuleOrder is the default set and application order of rules when the
// config does not specify a "rules" list.
var DefaultRuleOrder = []string{
	nameTrailingWhitespace,
	nameSetextHeadings,
	nameATXHeadings,
	nameListMarkers,
	nameSemBr,
	nameTableWidth,
	nameBlankLines,
}

// Build constructs an [Engine] from Viper config. It reads the ordered "rules"
// list (falling back to [DefaultRuleOrder]) and each rule's "options.<name>"
// sub-tree.
func Build(v *viper.Viper) (*Engine, error) {
	names := v.GetStringSlice("rules")
	if len(names) == 0 {
		names = DefaultRuleOrder
	}

	optsRoot := v.Sub("options")

	rules := make([]Rule, 0, len(names))
	for _, name := range names {
		f, ok := registry[name]
		if !ok {
			return nil, fmt.Errorf("unknown rule %q", name)
		}
		var sub *viper.Viper
		if optsRoot != nil {
			sub = optsRoot.Sub(name)
		}
		r, err := f(sub)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", name, err)
		}
		rules = append(rules, r)
	}
	return &Engine{rules: rules}, nil
}

// RegisteredRules returns the names of all registered rules (unordered). Useful
// for validation and documentation.
func RegisteredRules() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
