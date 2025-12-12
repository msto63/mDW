// Package translation provides prompt translation for language-agnostic intent analysis
package translation

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/msto63/mDW/internal/aristoteles/pipeline"
	"github.com/msto63/mDW/pkg/core/logging"
)

// LLMFunc is a function type for calling the LLM
type LLMFunc func(ctx context.Context, model string, systemPrompt string, userPrompt string) (string, error)

// Stage is the translation pipeline stage
type Stage struct {
	llmFunc LLMFunc
	model   string
	logger  *logging.Logger
}

// Config holds translation stage configuration
type Config struct {
	Model string
}

// DefaultConfig returns default translation stage configuration
func DefaultConfig() *Config {
	return &Config{
		Model: "llama3.2:3b", // Fast model for translation
	}
}

// NewStage creates a new translation stage
func NewStage(cfg *Config) *Stage {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Stage{
		model:  cfg.Model,
		logger: logging.New("aristoteles-translation"),
	}
}

// SetLLMFunc sets the LLM function for translation
func (s *Stage) SetLLMFunc(fn LLMFunc) {
	s.llmFunc = fn
}

// Name returns the stage name
func (s *Stage) Name() string {
	return "translation"
}

// Execute runs the translation stage
func (s *Stage) Execute(ctx context.Context, pctx *pipeline.Context) error {
	start := time.Now()

	// Detect source language
	sourceLang := detectLanguage(pctx.Prompt)
	pctx.SourceLanguage = sourceLang

	// If already English or no LLM available, use original prompt
	if sourceLang == "en" || s.llmFunc == nil {
		pctx.PromptForAnalysis = pctx.Prompt
		s.logger.Debug("Skipping translation",
			"reason", map[bool]string{true: "already_english", false: "no_llm"}[sourceLang == "en"],
			"language", sourceLang)
		return nil
	}

	// Translate to English for intent analysis
	translated, err := s.translateToEnglish(ctx, pctx.Prompt)
	if err != nil {
		// Fallback to original prompt on error
		s.logger.Warn("Translation failed, using original prompt", "error", err)
		pctx.PromptForAnalysis = pctx.Prompt
		return nil // Don't fail the pipeline
	}

	pctx.PromptForAnalysis = translated

	s.logger.Debug("Prompt translated for analysis",
		"source_language", sourceLang,
		"original_length", len(pctx.Prompt),
		"translated_length", len(translated),
		"duration", time.Since(start))

	return nil
}

// translateToEnglish translates the prompt to English using the LLM
func (s *Stage) translateToEnglish(ctx context.Context, prompt string) (string, error) {
	systemPrompt := `You are a translator. Translate the following text to English.
Output ONLY the translation, nothing else. Do not add explanations or comments.
If the text is already in English, output it unchanged.`

	response, err := s.llmFunc(ctx, s.model, systemPrompt, prompt)
	if err != nil {
		return "", err
	}

	// Clean up response (remove potential quotes or extra whitespace)
	translated := strings.TrimSpace(response)
	translated = strings.Trim(translated, "\"'")

	return translated, nil
}

// detectLanguage detects the language of the text using heuristics
// Returns ISO 639-1 language code (e.g., "en", "de", "fr")
func detectLanguage(text string) string {
	lower := strings.ToLower(text)

	// German indicators
	germanIndicators := []string{
		"ä", "ö", "ü", "ß", // Umlauts and special chars
		" der ", " die ", " das ", " und ", " ist ", " sind ", " ein ", " eine ",
		" mit ", " für ", " auf ", " von ", " zu ", " im ", " am ", " an ",
		" nicht ", " ich ", " du ", " wir ", " sie ", " er ", " es ",
		" wie ", " was ", " wer ", " wo ", " wann ", " warum ",
		" bitte ", " danke ", " hallo ", " guten ",
	}
	germanCount := 0
	for _, indicator := range germanIndicators {
		if strings.Contains(lower, indicator) {
			germanCount++
		}
	}

	// French indicators
	frenchIndicators := []string{
		"é", "è", "ê", "ë", "à", "â", "ù", "û", "î", "ï", "ô", "ç", "œ",
		" le ", " la ", " les ", " un ", " une ", " des ",
		" est ", " sont ", " avec ", " pour ", " dans ", " sur ",
		" je ", " tu ", " nous ", " vous ", " ils ", " elles ",
		" que ", " qui ", " quoi ", " où ", " quand ", " pourquoi ",
		" s'il ", " c'est ", " n'est ",
	}
	frenchCount := 0
	for _, indicator := range frenchIndicators {
		if strings.Contains(lower, indicator) {
			frenchCount++
		}
	}

	// Spanish indicators
	spanishIndicators := []string{
		"ñ", "¿", "¡", "á", "é", "í", "ó", "ú",
		" el ", " la ", " los ", " las ", " un ", " una ",
		" es ", " son ", " con ", " para ", " en ", " por ",
		" yo ", " tú ", " nosotros ", " ellos ", " ellas ",
		" que ", " qué ", " quién ", " dónde ", " cuándo ", " cómo ",
	}
	spanishCount := 0
	for _, indicator := range spanishIndicators {
		if strings.Contains(lower, indicator) {
			spanishCount++
		}
	}

	// Determine language based on indicator counts
	maxCount := germanCount
	detectedLang := "de"

	if frenchCount > maxCount {
		maxCount = frenchCount
		detectedLang = "fr"
	}
	if spanishCount > maxCount {
		maxCount = spanishCount
		detectedLang = "es"
	}

	// If no strong indicators found, check if it's ASCII-only (likely English)
	if maxCount < 2 {
		if isASCIIText(text) {
			return "en"
		}
	}

	// Default to English if no indicators found
	if maxCount == 0 {
		return "en"
	}

	return detectedLang
}

// isASCIIText checks if the text contains only ASCII characters
func isASCIIText(text string) bool {
	for _, r := range text {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}
