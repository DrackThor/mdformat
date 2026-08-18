package format

import (
	"strings"

	"github.com/spf13/viper"
)

const nameATXHeadings = "atx-headings"

func init() { Register(nameATXHeadings, newATXHeadings) }

// atxHeadings normalizes ATX headings: it removes leading indentation, ensures
// exactly one space between the leading "#" run and the heading text, and strips
// any optional closing "#" sequence.
type atxHeadings struct{}

func newATXHeadings(_ *viper.Viper) (Rule, error) { return atxHeadings{}, nil }

func (atxHeadings) Name() string { return nameATXHeadings }

func (atxHeadings) Apply(lines []string) ([]string, error) {
	mask := codeMask(lines)
	out := make([]string, len(lines))
	for i, l := range lines {
		if mask[i] {
			out[i] = l
			continue
		}
		if hashes, text, ok := parseATXHeading(l); ok {
			if text == "" {
				out[i] = hashes
			} else {
				out[i] = hashes + " " + text
			}
			continue
		}
		out[i] = l
	}
	return out, nil
}

// parseATXHeading reports whether l is an ATX heading and returns its "#" run
// and the trimmed heading text (with any closing "#" sequence removed).
func parseATXHeading(l string) (hashes, text string, ok bool) {
	s := strings.TrimLeft(l, " ")
	if len(l)-len(s) > 3 { // more than 3 leading spaces = not a heading
		return "", "", false
	}
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return "", "", false
	}
	rest := s[n:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return "", "", false // "#text" is not a heading per CommonMark
	}
	text = strings.TrimSpace(rest)
	// Strip an optional closing run of '#' only when it is preceded by a space
	// (CommonMark), so "# C#" keeps its trailing '#'.
	if j := strings.LastIndexFunc(text, func(r rune) bool { return r != '#' }); j >= 0 {
		if text[j] == ' ' || text[j] == '\t' {
			text = strings.TrimRight(text[:j], " \t")
		}
	} else if text != "" { // text is all '#'
		text = ""
	}
	return s[:n], text, true
}
