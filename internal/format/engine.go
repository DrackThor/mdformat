package format

import (
	"slices"
	"strings"
)

// Engine applies a fixed, ordered list of rules to a document.
type Engine struct {
	rules []Rule
}

// Rules returns the ordered rules the engine will apply.
func (e *Engine) Rules() []Rule { return e.rules }

// RuleChange records what one rule did to the document. Lines holds the 1-based
// numbers of the lines the rule rewrote when it kept the line count; when it did
// not, From and To bound the region it rewrote instead. Both are in the
// coordinates of that rule's own output, so a rule that runs later and inserts
// or removes lines shifts them.
type RuleChange struct {
	Rule     string
	Lines    []int
	From, To int
}

// Trace lists the rules that changed the document, in the order they ran. Rules
// that left it untouched are not recorded.
type Trace []RuleChange

// Format applies all rules in order to src and returns the formatted document.
// Line endings are normalized to LF and the result ends with exactly one
// newline (unless the document is empty).
func (e *Engine) Format(src []byte) ([]byte, error) {
	out, _, err := e.FormatTrace(src)
	return out, err
}

// FormatTrace formats src like [Engine.Format] and additionally reports which
// rules changed the document and where.
//
// For example, formatting "a \nb\nc \n" with only trailing-whitespace enabled
// returns one [RuleChange] naming that rule with Lines [1 3]. Rules that leave
// the document untouched produce no entry at all.
//
// See TestEngine_FormatTrace_ExactLines, TestEngine_FormatTrace_Range and
// TestEngine_FormatTrace_UnchangedRulesAbsent.
func (e *Engine) FormatTrace(src []byte) ([]byte, Trace, error) {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Split into lines without the trailing empty element produced by a final
	// newline, so rules see content lines only.
	lines := splitLines(text)

	var trace Trace
	for _, r := range e.rules {
		out, err := r.Apply(lines)
		if err != nil {
			return nil, nil, err
		}
		if !slices.Equal(lines, out) {
			change := RuleChange{Rule: r.Name()}
			change.Lines, change.From, change.To = diffLines(lines, out)
			trace = append(trace, change)
		}
		lines = out
	}

	if len(lines) == 0 {
		return []byte{}, trace, nil
	}
	return []byte(strings.Join(lines, "\n") + "\n"), trace, nil
}

// diffLines describes how after differs from before. Line counts that match give
// exact line numbers; when they do not, the common prefix and suffix are trimmed
// and the span between them is reported as a range, which is as much as can be
// said without a real diff.
//
// For example ["a" "b"] to ["a" "c"] reports line 2, while ["a" "b" "c"] to
// ["a" "c"] reports the range 2-2: the deleted line leaves nothing to point at,
// so the range collapses onto the line now at the seam.
//
// See TestDiffLines.
func diffLines(before, after []string) (lines []int, from, to int) {
	if len(before) == len(after) {
		for i := range before {
			if before[i] != after[i] {
				lines = append(lines, i+1)
			}
		}
		return lines, 0, 0
	}

	prefix := 0
	for prefix < len(before) && prefix < len(after) && before[prefix] == after[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(before)-prefix && suffix < len(after)-prefix &&
		before[len(before)-1-suffix] == after[len(after)-1-suffix] {
		suffix++
	}

	from = prefix + 1
	to = len(after) - suffix
	if to < from {
		to = from
	}
	return nil, from, to
}

// splitLines splits text on LF, dropping a single trailing empty line that a
// final newline would create.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	return lines
}
