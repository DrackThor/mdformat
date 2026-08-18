package format

import (
	"strings"

	"github.com/spf13/viper"
)

const nameListMarkers = "list-markers"

func init() { Register(nameListMarkers, newListMarkers) }

// listMarkers normalizes unordered list bullets to a single configured marker
// character ("-" by default). Ordered lists and thematic breaks are left alone.
type listMarkers struct {
	marker byte
}

func newListMarkers(opts *viper.Viper) (Rule, error) {
	m := "-"
	if opts != nil && opts.IsSet("marker") {
		if s := opts.GetString("marker"); s != "" {
			m = s
		}
	}
	return listMarkers{marker: m[0]}, nil
}

func (listMarkers) Name() string { return nameListMarkers }

func (r listMarkers) Apply(lines []string) ([]string, error) {
	mask := codeMask(lines)
	out := make([]string, len(lines))
	for i, l := range lines {
		if mask[i] || isThematicBreak(l) {
			out[i] = l
			continue
		}
		indent := len(l) - len(strings.TrimLeft(l, " "))
		if indent > 3 {
			out[i] = l // indented too far to be a list item at this level
			continue
		}
		rest := l[indent:]
		if len(rest) >= 2 && (rest[0] == '*' || rest[0] == '+' || rest[0] == '-') &&
			(rest[1] == ' ' || rest[1] == '\t') {
			out[i] = l[:indent] + string(r.marker) + rest[1:]
			continue
		}
		out[i] = l
	}
	return out, nil
}

// isThematicBreak reports whether l is a thematic break: three or more of the
// same "-", "*", or "_" character, optionally separated by spaces.
func isThematicBreak(l string) bool {
	s := strings.TrimSpace(l)
	if len(s) < 3 {
		return false
	}
	var ch byte
	count := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t':
			continue
		case '-', '*', '_':
			if ch == 0 {
				ch = s[i]
			} else if s[i] != ch {
				return false
			}
			count++
		default:
			return false
		}
	}
	return count >= 3
}
