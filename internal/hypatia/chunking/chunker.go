package chunking

import (
	"strings"
	"unicode"
)

// Chunk represents a text chunk
type Chunk struct {
	ID       string
	Content  string
	Index    int
	Start    int
	End      int
	Metadata map[string]string
}

// Strategy defines the chunking strategy
type Strategy string

const (
	StrategyFixed      Strategy = "fixed"
	StrategySentence   Strategy = "sentence"
	StrategyParagraph  Strategy = "paragraph"
	StrategyRecursive  Strategy = "recursive"
)

// Config holds chunking configuration
type Config struct {
	Strategy    Strategy
	ChunkSize   int
	ChunkOverlap int
	Separators  []string
}

// DefaultConfig returns default chunking configuration
func DefaultConfig() Config {
	return Config{
		Strategy:    StrategyRecursive,
		ChunkSize:   1000,
		ChunkOverlap: 200,
		Separators:  []string{"\n\n", "\n", ". ", " "},
	}
}

// Chunker splits text into chunks
type Chunker struct {
	config Config
}

// NewChunker creates a new chunker
func NewChunker(cfg Config) *Chunker {
	return &Chunker{config: cfg}
}

// Split splits text into chunks
func (c *Chunker) Split(text string, docID string) []Chunk {
	switch c.config.Strategy {
	case StrategyFixed:
		return c.splitFixed(text, docID)
	case StrategySentence:
		return c.splitSentence(text, docID)
	case StrategyParagraph:
		return c.splitParagraph(text, docID)
	case StrategyRecursive:
		return c.splitRecursive(text, docID)
	default:
		return c.splitFixed(text, docID)
	}
}

// splitFixed splits text into fixed-size chunks
func (c *Chunker) splitFixed(text string, docID string) []Chunk {
	var chunks []Chunk
	runes := []rune(text)
	length := len(runes)

	for i := 0; i < length; i += c.config.ChunkSize - c.config.ChunkOverlap {
		end := i + c.config.ChunkSize
		if end > length {
			end = length
		}

		chunk := Chunk{
			ID:      generateChunkID(docID, len(chunks)),
			Content: string(runes[i:end]),
			Index:   len(chunks),
			Start:   i,
			End:     end,
		}
		chunks = append(chunks, chunk)

		if end >= length {
			break
		}
	}

	return chunks
}

// splitSentence splits text by sentences
func (c *Chunker) splitSentence(text string, docID string) []Chunk {
	sentences := splitIntoSentences(text)
	return c.mergeSmallChunks(sentences, docID)
}

// splitParagraph splits text by paragraphs
func (c *Chunker) splitParagraph(text string, docID string) []Chunk {
	paragraphs := strings.Split(text, "\n\n")
	var cleaned []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return c.mergeSmallChunks(cleaned, docID)
}

// splitRecursive recursively splits text using multiple separators
func (c *Chunker) splitRecursive(text string, docID string) []Chunk {
	return c.recursiveSplit(text, docID, 0)
}

func (c *Chunker) recursiveSplit(text string, docID string, sepIndex int) []Chunk {
	if len(text) <= c.config.ChunkSize {
		return []Chunk{{
			ID:      generateChunkID(docID, 0),
			Content: text,
			Index:   0,
		}}
	}

	if sepIndex >= len(c.config.Separators) {
		// Fall back to fixed splitting
		return c.splitFixed(text, docID)
	}

	separator := c.config.Separators[sepIndex]
	parts := strings.Split(text, separator)

	var chunks []Chunk
	var currentChunk strings.Builder
	chunkIndex := 0

	for i, part := range parts {
		// Add separator back (except for first part)
		testContent := currentChunk.String()
		if testContent != "" {
			testContent += separator
		}
		testContent += part

		if len(testContent) > c.config.ChunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			content := currentChunk.String()
			if len(content) > c.config.ChunkSize {
				// Recursively split
				subChunks := c.recursiveSplit(content, docID, sepIndex+1)
				for _, sc := range subChunks {
					sc.Index = chunkIndex
					sc.ID = generateChunkID(docID, chunkIndex)
					chunks = append(chunks, sc)
					chunkIndex++
				}
			} else {
				chunks = append(chunks, Chunk{
					ID:      generateChunkID(docID, chunkIndex),
					Content: content,
					Index:   chunkIndex,
				})
				chunkIndex++
			}

			// Start new chunk with overlap
			currentChunk.Reset()
			if c.config.ChunkOverlap > 0 && len(content) > c.config.ChunkOverlap {
				overlap := content[len(content)-c.config.ChunkOverlap:]
				currentChunk.WriteString(overlap)
				currentChunk.WriteString(separator)
			}
		}

		if currentChunk.Len() > 0 && i > 0 {
			currentChunk.WriteString(separator)
		}
		currentChunk.WriteString(part)
	}

	// Don't forget the last chunk
	if currentChunk.Len() > 0 {
		content := currentChunk.String()
		if len(content) > c.config.ChunkSize {
			subChunks := c.recursiveSplit(content, docID, sepIndex+1)
			for _, sc := range subChunks {
				sc.Index = chunkIndex
				sc.ID = generateChunkID(docID, chunkIndex)
				chunks = append(chunks, sc)
				chunkIndex++
			}
		} else {
			chunks = append(chunks, Chunk{
				ID:      generateChunkID(docID, chunkIndex),
				Content: content,
				Index:   chunkIndex,
			})
		}
	}

	return chunks
}

// mergeSmallChunks merges small text segments into appropriately sized chunks
func (c *Chunker) mergeSmallChunks(segments []string, docID string) []Chunk {
	var chunks []Chunk
	var currentChunk strings.Builder
	chunkIndex := 0

	for _, segment := range segments {
		if currentChunk.Len()+len(segment)+1 > c.config.ChunkSize {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, Chunk{
					ID:      generateChunkID(docID, chunkIndex),
					Content: currentChunk.String(),
					Index:   chunkIndex,
				})
				chunkIndex++
				currentChunk.Reset()
			}
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(segment)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			ID:      generateChunkID(docID, chunkIndex),
			Content: currentChunk.String(),
			Index:   chunkIndex,
		})
	}

	return chunks
}

// splitIntoSentences splits text into sentences
func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		current.WriteRune(r)

		// Check for sentence endings
		if r == '.' || r == '!' || r == '?' {
			// Check if followed by space or end
			if i+1 >= len(runes) || unicode.IsSpace(runes[i+1]) {
				// Check it's not an abbreviation (simple heuristic)
				sentence := strings.TrimSpace(current.String())
				if len(sentence) > 0 {
					sentences = append(sentences, sentence)
					current.Reset()
				}
			}
		}
	}

	// Don't forget remaining text
	if current.Len() > 0 {
		sentence := strings.TrimSpace(current.String())
		if len(sentence) > 0 {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// generateChunkID generates a unique chunk ID
func generateChunkID(docID string, index int) string {
	return strings.ReplaceAll(docID, " ", "_") + "_chunk_" + itoa(index)
}

// itoa converts int to string (simple implementation)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
