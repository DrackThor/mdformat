package format

import "strings"

// frontMatterDelim opens and closes a leading front matter block.
const frontMatterDelim = "---"

// codeMask returns a slice parallel to lines where an entry is true when that
// line must be preserved verbatim by prose-oriented rules: YAML/TOML front
// matter (a leading fenced "---" block) and fenced code blocks (``` or ~~~).
func codeMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	start := frontMatterEnd(lines)
	for i := 0; i < start; i++ {
		mask[i] = true
	}
	for _, b := range fenceBlocks(lines, start) {
		last := b.close
		if last < 0 { // an unclosed fence runs to the end of the document
			last = len(lines) - 1
		}
		for i := b.open; i <= last; i++ {
			mask[i] = true
		}
	}
	return mask
}

// frontMatterEnd returns the index of the first line after a leading front
// matter block: a "---" on the very first line opens a metadata block that
// closes on the next "---" or "...". It returns 0 when there is no front
// matter, and len(lines) when the block is never closed.
func frontMatterEnd(lines []string) int {
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != frontMatterDelim {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if t := strings.TrimRight(lines[i], " \t"); t == frontMatterDelim || t == "..." {
			return i + 1
		}
	}
	return len(lines)
}

// fenceBlock is one fenced code block, split into the parts a rule may rewrite.
type fenceBlock struct {
	// open is the index of the opening line, close the index of the closing
	// line, or -1 when the fence is never closed.
	open, close int
	// indent is the whitespace before the opening marker, marker the opening
	// run itself, and info the rest of the opening line (the info string).
	indent, marker, info string
}

// fenceBlocks returns the fenced code blocks found from line start onward, in
// document order and never overlapping. Pass [frontMatterEnd] as start so a
// "---" front matter block is not mistaken for content.
func fenceBlocks(lines []string, start int) []fenceBlock {
	var blocks []fenceBlock
	for i := start; i < len(lines); i++ {
		indent, body := splitIndent(lines[i])
		marker := fenceMarker(body)
		if marker == "" {
			continue
		}
		b := fenceBlock{
			open:   i,
			close:  -1,
			indent: indent,
			marker: marker,
			info:   body[len(marker):],
		}
		// A closing fence uses the same character and is at least as long as
		// the opener.
		for j := i + 1; j < len(lines); j++ {
			m := fenceMarker(strings.TrimLeft(lines[j], " \t"))
			if m != "" && m[0] == marker[0] && len(m) >= len(marker) {
				b.close = j
				break
			}
		}
		blocks = append(blocks, b)
		if b.close < 0 {
			break
		}
		i = b.close
	}
	return blocks
}
