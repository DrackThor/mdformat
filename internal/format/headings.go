package format

import (
	"strings"
	"unicode/utf8"

	"github.com/spf13/viper"
)

const nameATXHeadings = "atx-headings"

// defaultHeadingPunctuation is the trailing punctuation stripped from heading
// text by default (markdownlint MD026).
const defaultHeadingPunctuation = ".,;:!?"

func init() { Register(nameATXHeadings, newATXHeadings) }

// atxHeadings normalizes ATX headings: it removes leading indentation, ensures
// exactly one space between the leading "#" run and the heading text, strips
// any optional closing "#" sequence, and trims trailing punctuation.
type atxHeadings struct {
	punctuation string
}

func newATXHeadings(opts *viper.Viper) (Rule, error) {
	r := atxHeadings{punctuation: defaultHeadingPunctuation}
	if opts != nil && opts.IsSet("strip-trailing-punctuation") {
		r.punctuation = opts.GetString("strip-trailing-punctuation")
	}
	return r, nil
}

func (atxHeadings) Name() string { return nameATXHeadings }

func (r atxHeadings) Apply(lines []string) ([]string, error) {
	mask := codeMask(lines)
	out := make([]string, len(lines))
	for i, l := range lines {
		if mask[i] {
			out[i] = l
			continue
		}
		if hashes, text, ok := parseATXHeading(l); ok {
			text = trimHeadingPunctuation(text, r.punctuation)
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

// trimHeadingPunctuation removes trailing runes of cutset from heading text. A
// mark kept literal by a backslash escape ("Done\.") is left in place, as
// dropping it would change what the heading renders.
func trimHeadingPunctuation(text, cutset string) string {
	for cutset != "" && text != "" {
		last, size := utf8.DecodeLastRuneInString(text)
		if !strings.ContainsRune(cutset, last) {
			break
		}
		head := text[:len(text)-size]
		if escapedEnd(head) {
			break
		}
		text = strings.TrimRight(head, " \t")
	}
	return text
}

// escapedEnd reports whether s ends in an odd number of backslashes, i.e. the
// rune that follows it is escaped.
func escapedEnd(s string) bool {
	n := 0
	for n < len(s) && s[len(s)-1-n] == '\\' {
		n++
	}
	return n%2 == 1
}
