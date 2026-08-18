package format

import (
	"strings"

	"github.com/spf13/viper"
)

const nameTrailingWhitespace = "trailing-whitespace"

func init() { Register(nameTrailingWhitespace, newTrailingWhitespace) }

// trailingWhitespace trims trailing spaces and tabs from each line. A Markdown
// hard line break (two or more trailing spaces on a non-blank line) is
// preserved as exactly two spaces. Lines inside verbatim spans are left as-is.
type trailingWhitespace struct{}

func newTrailingWhitespace(_ *viper.Viper) (Rule, error) { return trailingWhitespace{}, nil }

func (trailingWhitespace) Name() string { return nameTrailingWhitespace }

func (trailingWhitespace) Apply(lines []string) ([]string, error) {
	mask := codeMask(lines)
	out := make([]string, len(lines))
	for i, l := range lines {
		if mask[i] {
			out[i] = l
			continue
		}
		trimmed := strings.TrimRight(l, " \t")
		// Preserve a hard line break: non-blank content followed by two or more
		// trailing spaces (a tab does not create a hard break).
		trailingSpaces := len(l) - len(strings.TrimRight(l, " "))
		if trimmed != "" && trailingSpaces >= 2 && !strings.HasSuffix(l, "\t") {
			out[i] = trimmed + "  "
			continue
		}
		out[i] = trimmed
	}
	return out, nil
}
