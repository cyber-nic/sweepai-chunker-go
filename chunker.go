package chunker

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func New(options ...func(*Chunker)) Chunker {
	c := Chunker{
		maxChunkSize:      1024 * 3, // 3KB
		coalesceThreshold: 50,
	}

	// Apply any additional options
	for _, o := range options {
		o(&c)
	}

	return c
}

// WithMaxChunkSize sets the maximum chunk size in bytes.
func WithMaxChunkSize(value int) func(*Chunker) {
	return func(o *Chunker) {
		o.maxChunkSize = value
	}
}

// WithCoalesce sets the minimum number of non-whitespace characters to finalize a chunk.
func WithCoalesceThreshold(value int) func(*Chunker) {
	return func(o *Chunker) {
		o.coalesceThreshold = value
	}
}

// Chunker is a struct that allows to
type Chunker struct {
	maxChunkSize      int
	coalesceThreshold int
}

// Chunk tracks a [Start, End) region and the text within it.
type Chunk struct {
	Start int
	End   int
	Text  string
}

// Span tracks a [Start, End) region. It is used for both byte offsets and line offsets.
type Span struct {
	Start int
	End   int
}

// extractByLine returns the text lines from [Span.Start : Span.End] in a given string.
func (sp Span) extractByLine(s string) string {
	lines := strings.Split(s, "\n")
	if sp.Start >= len(lines) {
		return ""
	}
	if sp.End > len(lines) {
		return strings.Join(lines[sp.Start:], "\n")
	}
	return strings.Join(lines[sp.Start:sp.End], "\n")
}

// length is simply (End - Start).
func (sp Span) length() int {
	return sp.End - sp.Start
}

// countLengthWithoutWhitespace removes all whitespace and returns the length.
func countLengthWithoutWhitespace(s string) int {
	re := regexp.MustCompile(`\s+`)
	return len(re.ReplaceAllString(s, ""))
}

// getLineNumber maps a byte index into the code to a 0-based line number.
func getLineNumber(idx int, content string) int {
	if idx < 0 {
		return 0
	}
	lines := strings.SplitAfter(content, "\n")
	total := 0
	for ln, line := range lines {
		total += len(line)
		if total > idx {
			return ln
		}
	}
	return len(lines)
}

// chunkenate recursively splits a file’s AST into chunks by byte offset.
// Then it merges small chunks and ultimately returns line-based spans.
func (c Chunker) chunkenate(rootNode *sitter.Node, source []byte) []Span {
	// chunkerHelper is the recursive function that collects child spans.
	var chunkerHelper func(node *sitter.Node, code []byte, startPos int) []Span
	chunkerHelper = func(node *sitter.Node, code []byte, startPos int) []Span {
		var spans []Span
		current := Span{Start: startPos, End: startPos}
		childCount := int(node.ChildCount())

		for i := 0; i < childCount; i++ {
			child := node.Child(uint(i))
			if child == nil {
				continue
			}
			childSpan := Span{
				Start: int(child.StartByte()),
				End:   int(child.EndByte()),
			}

			// If the child itself exceeds maxChunkSize, recursively chunk it.
			if childSpan.length() > c.maxChunkSize {
				spans = append(spans, current)
				sub := chunkerHelper(child, code, childSpan.Start)
				spans = append(spans, sub...)
				current = Span{
					Start: childSpan.End,
					End:   childSpan.End,
				}
			} else if current.length()+childSpan.length() > c.maxChunkSize {
				// End the current chunk and start a new one.
				spans = append(spans, current)
				current = childSpan
			} else {
				// Extend the current chunk’s End to childSpan.End
				current.End = childSpan.End
			}
		}
		if current.length() > 0 {
			spans = append(spans, current)
		}
		return spans
	}

	// 1. Initial pass: chunk by bytes
	spans := chunkerHelper(rootNode, source, 0)

	// 2. "Remove gaps": set each chunk’s End to the next chunk’s Start
	for i := 0; i < len(spans)-1; i++ {
		spans[i].End = spans[i+1].Start
	}

	// 3. "Combine small chunks" based on coalesce threshold
	var newChunks []Span
	var current Span
	for i := 0; i < len(spans); i++ {
		if current.length() == 0 {
			current = spans[i]
		} else {
			current.End = spans[i].End
		}
		snippet := source[current.Start:current.End]
		if countLengthWithoutWhitespace(string(snippet)) > c.coalesceThreshold &&
			bytes.ContainsRune(snippet, '\n') {
			newChunks = append(newChunks, current)
			current = Span{
				Start: spans[i].End,
				End:   spans[i].End,
			}
		}
	}
	if current.length() > 0 {
		newChunks = append(newChunks, current)
	}

	// 4. Convert final byte spans to line-based spans
	var lineChunks []Span
	for _, sp := range newChunks {
		startLine := getLineNumber(sp.Start, string(source))
		endLine := getLineNumber(sp.End, string(source))
		if endLine > startLine {
			lineChunks = append(lineChunks, Span{Start: startLine, End: endLine})
		}
	}

	return lineChunks
}

// Harvest is the higher-level function that:
// 1) Parses the code using the Go grammar
// 2) Runs the chunker logic
// 3) Converts line spans to actual text chunks
func (c Chunker) Harvest(filePath string) ([]Chunk, error) {
	// Read the file content.
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", filePath, err)
	}

	// Get the language from the filename.
	// Determines the programming language to use for parsing based on the file extension.
	lang, _, err := GetLanguageFromFileName(filePath)
	if err != nil {
		return nil, fmt.Errorf("error getting language from filename: %v", err)
	}

	// Return an error if the language is not supported.
	if lang == nil {
		return nil, fmt.Errorf("unrecognized or unsupported file type (%s)", filePath)
	}

	// Initialize Tree-sitter parser for parsing source code into an abstract syntax tree (AST).
	parser := sitter.NewParser()
	parser.SetLanguage(lang) // Set the parser's language to match the file type.

	tree := parser.Parse(content, nil)
	if tree == nil || tree.RootNode() == nil {
		return nil, fmt.Errorf("tree-sitter failed to parse %s", filePath)
	}

	lineSpans := c.chunkenate(tree.RootNode(), content)

	chunks := []Chunk{}
	for _, sp := range lineSpans {
		// t := sp.extractByLine(string(content))
		// fmt.Printf(">>%s<<", t)
		// if strings.TrimSpace(t) == "" {
		// 	continue
		// }

		chunks = append(chunks, Chunk{
			Start: sp.Start,
			End:   sp.End,
			Text:  sp.extractByLine(string(content)),
		})
	}

	return chunks, nil
}
