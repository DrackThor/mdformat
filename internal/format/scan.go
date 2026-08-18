package format

import "strings"

// frontMatterDelim opens and closes a leading front matter block.
const frontMatterDelim = "---"

// codeMask returns a slice parallel to lines where an entry is true when that
// line must be preserved verbatim by prose-oriented rules: YAML/TOML front
// matter (a leading fenced "---" block) and fenced code blocks (``` or ~~~).
func codeMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	i := 0

	// Front matter: a "---" on the very first line opens a metadata block that
	// closes on the next "---" or "...".
	if len(lines) > 0 && strings.TrimRight(lines[0], " \t") == frontMatterDelim {
		mask[0] = true
		for i = 1; i < len(lines); i++ {
			mask[i] = true
			if t := strings.TrimRight(lines[i], " \t"); t == frontMatterDelim || t == "..." {
				i++
				break
			}
		}
	}

	fence := "" // "" when outside a fence, else the opening marker run
	for ; i < len(lines); i++ {
		marker := fenceMarker(strings.TrimLeft(lines[i], " \t"))
		if fence == "" {
			if marker != "" {
				fence = marker
				mask[i] = true
			}
			continue
		}
		// Inside a fence: every line is verbatim; a closing fence uses the same
		// character and is at least as long as the opener.
		mask[i] = true
		if marker != "" && marker[0] == fence[0] && len(marker) >= len(fence) {
			fence = ""
		}
	}
	return mask
}

// fenceMarker returns the leading run of backticks or tildes (length >= 3) that
// starts s, or "" if s does not open/close a code fence.
func fenceMarker(s string) string {
	if !strings.HasPrefix(s, "```") && !strings.HasPrefix(s, "~~~") {
		return ""
	}
	ch := s[0]
	n := 0
	for n < len(s) && s[n] == ch {
		n++
	}
	return s[:n]
}
