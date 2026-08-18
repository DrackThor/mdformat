package format

import (
	"strings"

	"github.com/spf13/viper"
)

const (
	nameSemBr      = "semantic-line-breaks"
	breakSentence  = "sentence"
	breakColon     = "colon"
	breakSemicolon = "semicolon"
	breakComma     = "comma"
)

func init() { Register(nameSemBr, newSemBr) }

// semBr applies semantic line breaks (https://sembr.org): it puts each sentence
// (and, optionally, each independent clause) on its own line. Breaks are
// inserted after configured punctuation, never inside inline code, links, or
// verbatim blocks, and continuation lines keep the block prefix (list marker
// indentation, blockquote markers) so rendering is unchanged.
//
// The default only breaks at sentence boundaries (. ! ?). Additional break
// points are enabled via the "break-on" option:
//
//	sentence   after . ! ?          (default)
//	colon      after :
//	semicolon  after ;
//	em-dash    after —
//	comma      after ,
type semBr struct {
	sentence, colon, semicolon, emDash, comma bool
}

func newSemBr(opts *viper.Viper) (Rule, error) {
	kinds := []string{breakSentence}
	if opts != nil && opts.IsSet("break-on") {
		kinds = opts.GetStringSlice("break-on")
	}
	r := semBr{}
	for _, k := range kinds {
		switch strings.ToLower(strings.TrimSpace(k)) {
		case breakSentence:
			r.sentence = true
		case breakColon:
			r.colon = true
		case breakSemicolon:
			r.semicolon = true
		case "em-dash", "emdash":
			r.emDash = true
		case breakComma:
			r.comma = true
		}
	}
	return r, nil
}

func (semBr) Name() string { return nameSemBr }

func (r semBr) Apply(lines []string) ([]string, error) {
	mask := codeMask(lines)
	out := make([]string, 0, len(lines))
	for i, l := range lines {
		if mask[i] || !r.isProse(l) {
			out = append(out, l)
			continue
		}
		content, contPrefix := splitPrefix(l)
		firstPrefix := l[:len(l)-len(content)]
		chunks := r.splitSentences(content)
		if len(chunks) <= 1 {
			out = append(out, l)
			continue
		}
		out = append(out, firstPrefix+chunks[0])
		for _, c := range chunks[1:] {
			out = append(out, contPrefix+c)
		}
	}
	return out, nil
}

// isProse reports whether a line's content is ordinary prose eligible for
// sentence splitting (not a heading, table row, HTML block, or thematic break).
func (semBr) isProse(l string) bool {
	t := strings.TrimSpace(l)
	if t == "" {
		return false
	}
	if strings.ContainsRune(t, '|') { // table rows are handled by table-width
		return false
	}
	content, _ := splitPrefix(l)
	c := strings.TrimSpace(content)
	if c == "" {
		return false
	}
	switch c[0] {
	case '#', '<', '>', '=': // heading, HTML/autolink block, quote-only, setext
		return false
	}
	return !isThematicBreak(l)
}

// splitPrefix separates a line's block prefix (indentation, blockquote markers,
// list marker) from its content. contPrefix is the prefix to place on
// continuation lines: blockquote markers are kept so text stays in the quote,
// and a list marker becomes equal-width spaces so text aligns under the item.
func splitPrefix(line string) (content, contPrefix string) {
	i := 0
	var b strings.Builder
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		b.WriteByte(line[i])
		i++
	}
	for i < len(line) && line[i] == '>' {
		b.WriteByte('>')
		i++
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			b.WriteByte(line[i])
			i++
		}
	}
	if m := listMarkerLen(line[i:]); m > 0 {
		b.WriteString(strings.Repeat(" ", m))
		i += m
	}
	return line[i:], b.String()
}

// listMarkerLen returns the byte length of a leading list marker in s
// (bullet "- ", "* ", "+ " or ordered "1. " / "1) "), including its trailing
// whitespace, or 0 if s does not start with one.
func listMarkerLen(s string) int {
	if s == "" {
		return 0
	}
	var i int
	switch s[0] {
	case '-', '*', '+':
		i = 1
	default:
		i = orderedMarkerLen(s)
		if i == 0 {
			return 0
		}
	}
	spaces := 0
	for i+spaces < len(s) && (s[i+spaces] == ' ' || s[i+spaces] == '\t') {
		spaces++
	}
	if spaces == 0 {
		return 0
	}
	return i + spaces
}

// orderedMarkerLen returns the byte length of a leading ordered-list marker
// ("123." or "123)") in s, or 0 if s does not start with one.
func orderedMarkerLen(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(s) || (s[i] != '.' && s[i] != ')') {
		return 0
	}
	return i + 1
}

// splitSentences splits content into chunks at enabled break punctuation,
// consuming the whitespace that followed the punctuation.
func (r semBr) splitSentences(content string) []string {
	prot := inlineProtected(content)
	var chunks []string
	start := 0
	i := 0
	for i < len(content) {
		width, isBreak := r.breakAt(content, i, prot)
		if !isBreak {
			i++
			continue
		}
		end := i + width // index just past the punctuation
		for end < len(content) && isCloser(content[end]) {
			end++
		}
		k := end
		for k < len(content) && content[k] == ' ' {
			k++
		}
		// Only break when whitespace and further content follow.
		if k > end && k < len(content) {
			chunks = append(chunks, strings.TrimRight(content[start:end], " "))
			start = k
			i = k
			continue
		}
		i += width
	}
	chunks = append(chunks, content[start:])
	return chunks
}

// breakAt reports whether a break point begins at content[i] and how many bytes
// the punctuation occupies.
func (r semBr) breakAt(content string, i int, prot []bool) (width int, ok bool) {
	if prot[i] {
		return 0, false
	}
	switch content[i] {
	case '.', '!', '?':
		if !r.sentence {
			return 0, false
		}
		if content[i] == '.' && !sentenceDot(content, i) {
			return 0, false
		}
		return 1, true
	case ':':
		return boolWidth(r.colon)
	case ';':
		return boolWidth(r.semicolon)
	case ',':
		return boolWidth(r.comma)
	}
	if r.emDash && strings.HasPrefix(content[i:], "—") {
		return len("—"), true
	}
	return 0, false
}

func boolWidth(enabled bool) (int, bool) {
	if enabled {
		return 1, true
	}
	return 0, false
}

// isCloser reports whether c is a closing quote or bracket that may follow
// sentence punctuation and should stay on the same line.
func isCloser(c byte) bool {
	switch c {
	case '"', '\'', ')', ']', '}', '`', '*', '_':
		return true
	}
	return false
}

// sentenceDot reports whether the '.' at index i ends a sentence rather than
// being a decimal point, ellipsis, initial, or known abbreviation.
func sentenceDot(content string, i int) bool {
	if i > 0 && content[i-1] == '.' { // part of "..."
		return false
	}
	if i+1 < len(content) && content[i+1] == '.' { // start of "..."
		return false
	}
	// Preceding word (letters and internal dots), e.g. "e.g" or "Mr".
	j := i
	for j > 0 && (isLetter(content[j-1]) || content[j-1] == '.') {
		j--
	}
	word := strings.ToLower(strings.Trim(content[j:i], "."))
	if word == "" {
		return true
	}
	if len(word) == 1 && content[j] >= 'A' && content[j] <= 'Z' {
		return false // single-letter initial, e.g. "J. Smith"
	}
	return !abbreviations[word]
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

var abbreviations = map[string]bool{
	"e.g": true, "i.e": true, "etc": true, "vs": true, "cf": true, "al": true,
	"mr": true, "mrs": true, "ms": true, "dr": true, "prof": true, "st": true,
	"jr": true, "sr": true, "inc": true, "ltd": true, "co": true, "no": true,
	"vol": true, "fig": true, "eq": true, "approx": true, "sec": true, "min": true,
}

// inlineProtected marks byte positions in s that lie inside spans where breaks
// must not occur: inline code spans (backticks), autolinks (<...>), and link or
// image destinations (the "(...)" after "]").
func inlineProtected(s string) []bool {
	p := make([]bool, len(s))
	i := 0
	for i < len(s) {
		switch {
		case s[i] == '`':
			n := runLen(s, i, '`')
			if end := findRun(s, i+n, '`', n); end >= 0 {
				markRange(p, i, end+n)
				i = end + n
				continue
			}
		case s[i] == '<':
			if end := strings.IndexByte(s[i:], '>'); end >= 0 && !strings.ContainsAny(s[i:i+end], " ") {
				markRange(p, i, i+end+1)
				i = i + end + 1
				continue
			}
		case s[i] == ']' && i+1 < len(s) && s[i+1] == '(':
			if end := strings.IndexByte(s[i+1:], ')'); end >= 0 {
				markRange(p, i+1, i+1+end+1)
				i = i + 1 + end + 1
				continue
			}
		}
		i++
	}
	return p
}

func runLen(s string, i int, ch byte) int {
	n := 0
	for i+n < len(s) && s[i+n] == ch {
		n++
	}
	return n
}

// findRun returns the start index of the first run of exactly n ch bytes at or
// after from, or -1.
func findRun(s string, from int, ch byte, n int) int {
	for i := from; i < len(s); i++ {
		if s[i] == ch && runLen(s, i, ch) == n {
			return i
		}
	}
	return -1
}

func markRange(p []bool, lo, hi int) {
	for i := lo; i < hi && i < len(p); i++ {
		p[i] = true
	}
}
