package format

import "strings"

// Engine applies a fixed, ordered list of rules to a document.
type Engine struct {
	rules []Rule
}

// Rules returns the ordered rules the engine will apply.
func (e *Engine) Rules() []Rule { return e.rules }

// Format applies all rules in order to src and returns the formatted document.
// Line endings are normalized to LF and the result ends with exactly one
// newline (unless the document is empty).
func (e *Engine) Format(src []byte) ([]byte, error) {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Split into lines without the trailing empty element produced by a final
	// newline, so rules see content lines only.
	lines := splitLines(text)

	for _, r := range e.rules {
		out, err := r.Apply(lines)
		if err != nil {
			return nil, err
		}
		lines = out
	}

	if len(lines) == 0 {
		return []byte{}, nil
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
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
