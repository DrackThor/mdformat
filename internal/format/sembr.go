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
// (and, optionally, each independent clause) on its own line. A paragraph is
// first unwrapped — hard-wrapped lines are stitched back into one logical line,
// so fixed-width formatted documents are reflowed rather than broken further —
// and then split again at the configured punctuation. Breaks are never inserted
// inside inline code, links, or verbatim blocks, and continuation lines keep the
// block prefix (list marker indentation, blockquote markers) so rendering is
// unchanged.
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
	for i := 0; i < len(lines); {
		if mask[i] || isIndented(lines[i]) || !isProse(lines[i]) {
			out = append(out, lines[i])
			i++
			continue
		}
		joined, next := r.unwrap(lines, mask, i)
		i = next

		content, contPrefix := splitPrefix(joined)
		firstPrefix := joined[:len(joined)-len(content)]
		for k, c := range r.splitSentences(content) {
			if k == 0 {
				out = append(out, firstPrefix+c)
				continue
			}
			out = append(out, contPrefix+c)
		}
	}
	return out, nil
}

// unwrap folds the wrapped continuation lines that follow lines[i] into a single
// logical line and returns it together with the index of the first line it did
// not consume.
func (r semBr) unwrap(lines []string, mask []bool, i int) (joined string, next int) {
	joined = lines[i]
	// prev is the last source line folded in, not the accumulation: joining
	// strips a line's trailing whitespace, so only the original still shows
	// whether it ended in a hard line break.
	prev := lines[i]
	for next = i + 1; next < len(lines); next++ {
		if mask[next] || !r.continues(prev, lines[next]) {
			break
		}
		content, _ := splitPrefix(lines[next])
		joined = strings.TrimRight(joined, " \t") + " " + strings.TrimSpace(content)
		if hardLineBreak(lines[next]) {
			joined += "  " // the fold ends here; keep the break visible
		}
		prev = lines[next]
	}
	return joined, next
}

// continues reports whether next is a wrapped continuation of the logical line
// prev and may therefore be folded into it. A hard line break, a change of
// blockquote depth, a new list item, and indented (verbatim) content all end the
// logical line.
func (r semBr) continues(prev, next string) bool {
	if hardLineBreak(prev) || !isProse(next) {
		return false
	}
	if quoteDepth(prev) != quoteDepth(next) {
		return false
	}
	if isIndented(prev) || isIndented(next) {
		return false
	}
	_, _, contentStart := scanPrefix(next)
	return listMarkerLen(next[contentStart:]) == 0
}

// hardLineBreak reports whether a line ends in a Markdown hard line break, which
// must survive unwrapping.
func hardLineBreak(line string) bool {
	return strings.HasSuffix(line, "  ") || strings.HasSuffix(line, "\\")
}

// quoteDepth returns the number of blockquote markers a line opens with.
func quoteDepth(line string) int {
	quoteEnd, _, _ := scanPrefix(line)
	return strings.Count(line[:quoteEnd], ">")
}

// isProse reports whether a line's content is ordinary prose eligible for
// sentence splitting. A list item qualifies: its marker is part of the block
// prefix, and the text after it is prose like any other.
func isProse(l string) bool {
	content, _ := splitPrefix(l)
	return classify(content) == kindParagraph
}

// splitPrefix separates a line's block prefix (indentation, blockquote markers,
// list marker) from its content. contPrefix is the prefix to place on
// continuation lines: blockquote markers are kept so text stays in the quote,
// and a list marker becomes equal-width spaces so text aligns under the item.
func splitPrefix(line string) (content, contPrefix string) {
	_, _, i := scanPrefix(line)
	contPrefix = line[:i]
	if m := listMarkerLen(line[i:]); m > 0 {
		contPrefix += strings.Repeat(" ", m)
		i += m
	}
	return line[i:], contPrefix
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
