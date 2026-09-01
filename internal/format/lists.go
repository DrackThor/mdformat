package format

import "github.com/spf13/viper"

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
		indent, rest := splitIndent(l)
		if indentWidth(indent) > 3 {
			out[i] = l // indented too far to be a list item at this level
			continue
		}
		// Ordered markers are left alone, so a bullet is required here even
		// though listMarkerLen accepts both.
		if listMarkerLen(rest) > 0 && (rest[0] == '*' || rest[0] == '+' || rest[0] == '-') {
			out[i] = indent + string(r.marker) + rest[1:]
			continue
		}
		out[i] = l
	}
	return out, nil
}
