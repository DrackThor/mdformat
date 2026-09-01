package format

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const nameCodeFenceStyle = "code-fence-style"

// defaultFenceMarker is the fence character used when the config does not pick
// one, and minFenceLen the shortest fence CommonMark accepts.
const (
	defaultFenceMarker = '`'
	minFenceLen        = 3
)

func init() { Register(nameCodeFenceStyle, newCodeFenceStyle) }

// codeFenceStyle rewrites every fenced code block with one marker character
// (markdownlint MD048) and the shortest valid fence length. A fence grows past
// minFenceLen only when its content holds a run of the target marker that would
// otherwise close the block early:
//
//	~~~        ````
//	```js  ->  ```js
//	```        ```
//	~~~        ````
//
// Indented code blocks (MD046) are left as they are.
type codeFenceStyle struct {
	marker byte
}

func newCodeFenceStyle(opts *viper.Viper) (Rule, error) {
	r := codeFenceStyle{marker: defaultFenceMarker}
	if opts != nil && opts.IsSet("marker") {
		m := opts.GetString("marker")
		if m != "`" && m != "~" {
			return nil, fmt.Errorf("marker %q: want %q or %q", m, "`", "~")
		}
		r.marker = m[0]
	}
	return r, nil
}

func (codeFenceStyle) Name() string { return nameCodeFenceStyle }

func (r codeFenceStyle) Apply(lines []string) ([]string, error) {
	blocks := fenceBlocks(lines, frontMatterEnd(lines))
	if len(blocks) == 0 {
		return lines, nil
	}
	out := make([]string, len(lines))
	copy(out, lines)

	for _, b := range blocks {
		// A backtick fence may not carry a backtick in its info string, so a
		// tilde fence that does keeps the marker it was written with.
		if r.marker == '`' && strings.ContainsRune(b.info, '`') {
			continue
		}
		fence := strings.Repeat(string(r.marker), r.fenceLen(lines, b))
		out[b.open] = b.indent + fence + b.info
		if b.close >= 0 {
			indent, rest := splitIndent(lines[b.close])
			out[b.close] = indent + fence + rest[len(fenceMarker(rest)):]
		}
	}
	return out, nil
}

// fenceLen returns the fence length block b needs: minFenceLen, unless its
// content starts a line with a longer run of the target marker, which would
// close the block early.
func (r codeFenceStyle) fenceLen(lines []string, b fenceBlock) int {
	end := b.close
	if end < 0 {
		end = len(lines)
	}
	longest := 0
	for i := b.open + 1; i < end; i++ {
		_, rest := splitIndent(lines[i])
		if run := runLen(rest, 0, r.marker); run > longest {
			longest = run
		}
	}
	if longest >= minFenceLen {
		return longest + 1
	}
	return minFenceLen
}
