package integration

import (
	"testing"
	"time"

	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	commonpb "github.com/msto63/mDW/api/gen/common"
)

func TestBabbage_HealthCheck(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	logTestStart(t, "Babbage", "HealthCheck")

	conn := dialGRPC(t, cfg.BabbageAddr)
	client := babbagepb.NewBabbageServiceClient(conn)

	ctx, cancel := testContext(t, 10*time.Second)
	defer cancel()

	resp, err := client.HealthCheck(ctx, &commonpb.HealthCheckRequest{})
	requireNoError(t, err, "HealthCheck failed")
	requireEqual(t, "healthy", resp.Status, "Service should be healthy")

	t.Logf("Babbage health: status=%s version=%s", resp.Status, resp.Version)
}

func TestBabbage_Analyze(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	logTestStart(t, "Babbage", "Analyze")

	conn := dialGRPC(t, cfg.BabbageAddr)
	client := babbagepb.NewBabbageServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	resp, err := client.Analyze(ctx, &babbagepb.AnalyzeRequest{
		Text: "Das ist ein wunderbarer Tag! Die Sonne scheint und die Vögel singen. Berlin ist eine tolle Stadt.",
	})
	requireNoError(t, err, "Analyze failed")
	requireNotEmpty(t, resp.Language, "Language should be detected")

	t.Logf("Analysis results:")
	t.Logf("  Language: %s (confidence: %.2f)", resp.Language, resp.LanguageConfidence)
	if resp.Sentiment != nil {
		t.Logf("  Sentiment: %s (score: %.2f)", resp.Sentiment.Sentiment, resp.Sentiment.Score)
	}
	t.Logf("  Keywords: %d found", len(resp.Keywords))
	t.Logf("  Entities: %d found", len(resp.Entities))
}

func TestBabbage_AnalyzeSentiment(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	logTestStart(t, "Babbage", "AnalyzeSentiment")

	conn := dialGRPC(t, cfg.BabbageAddr)
	client := babbagepb.NewBabbageServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	tests := []struct {
		text     string
		expected babbagepb.Sentiment
	}{
		{"Das ist fantastisch! Ich bin so glücklich!", babbagepb.Sentiment_SENTIMENT_POSITIVE},
		{"Das ist schrecklich. Ich bin sehr enttäuscht.", babbagepb.Sentiment_SENTIMENT_NEGATIVE},
		{"Die Temperatur beträgt 20 Grad.", babbagepb.Sentiment_SENTIMENT_NEUTRAL},
	}

	for _, tc := range tests {
		t.Run(tc.expected.String(), func(t *testing.T) {
			resp, err := client.AnalyzeSentiment(ctx, &babbagepb.SentimentRequest{
				Text: tc.text,
			})
			requireNoError(t, err, "AnalyzeSentiment failed")

			t.Logf("Text: '%s'", tc.text)
			if resp.Result != nil {
				t.Logf("  Sentiment: %s (score: %.2f, confidence: %.2f)",
					resp.Result.Sentiment, resp.Result.Score, resp.Result.Confidence)
			}
		})
	}
}

func TestBabbage_ExtractKeywords(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	logTestStart(t, "Babbage", "ExtractKeywords")

	conn := dialGRPC(t, cfg.BabbageAddr)
	client := babbagepb.NewBabbageServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	resp, err := client.ExtractKeywords(ctx, &babbagepb.ExtractRequest{
		Text: "Künstliche Intelligenz und maschinelles Lernen revolutionieren die Softwareentwicklung und Datenanalyse.",
	})
	requireNoError(t, err, "ExtractKeywords failed")
	requireTrue(t, len(resp.Keywords) > 0, "Should extract at least one keyword")

	t.Logf("Extracted %d keywords:", len(resp.Keywords))
	for _, kw := range resp.Keywords {
		t.Logf("  - %s (score: %.2f, freq: %d)", kw.Word, kw.Score, kw.Frequency)
	}
}

func TestBabbage_ExtractEntities(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	logTestStart(t, "Babbage", "ExtractEntities")

	conn := dialGRPC(t, cfg.BabbageAddr)
	client := babbagepb.NewBabbageServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	resp, err := client.ExtractEntities(ctx, &babbagepb.ExtractRequest{
		Text: "Angela Merkel war Bundeskanzlerin von Deutschland. Sie studierte an der Universität Leipzig.",
	})
	requireNoError(t, err, "ExtractEntities failed")

	t.Logf("Extracted %d entities:", len(resp.Entities))
	for _, e := range resp.Entities {
		t.Logf("  - %s (%s, confidence: %.2f)", e.Text, e.Type, e.Confidence)
	}
}

func TestBabbage_DetectLanguage(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	logTestStart(t, "Babbage", "DetectLanguage")

	conn := dialGRPC(t, cfg.BabbageAddr)
	client := babbagepb.NewBabbageServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	tests := []struct {
		text         string
		expectedLang string
	}{
		{"Dies ist ein deutscher Text.", "de"},
		{"This is an English text.", "en"},
		{"Ceci est un texte français.", "fr"},
		{"Este es un texto en español.", "es"},
	}

	for _, tc := range tests {
		t.Run(tc.expectedLang, func(t *testing.T) {
			resp, err := client.DetectLanguage(ctx, &babbagepb.DetectLanguageRequest{
				Text: tc.text,
			})
			requireNoError(t, err, "DetectLanguage failed")

			t.Logf("Text: '%s'", tc.text)
			t.Logf("  Language: %s (confidence: %.2f)", resp.Language, resp.Confidence)

			if len(resp.Alternatives) > 0 {
				t.Logf("  Alternatives:")
				for _, alt := range resp.Alternatives {
					t.Logf("    - %s (%.2f)", alt.Language, alt.Score)
				}
			}
		})
	}
}

func TestBabbage_Summarize(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	logTestStart(t, "Babbage", "Summarize")

	conn := dialGRPC(t, cfg.BabbageAddr)
	client := babbagepb.NewBabbageServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	longText := `Die künstliche Intelligenz hat in den letzten Jahren enorme Fortschritte gemacht.
	Maschinelles Lernen ermöglicht es Computern, aus Daten zu lernen und Muster zu erkennen.
	Deep Learning nutzt neuronale Netze mit vielen Schichten für komplexe Aufgaben.
	Natürliche Sprachverarbeitung erlaubt die Kommunikation zwischen Menschen und Maschinen.
	Computer Vision ermöglicht das Verstehen und Analysieren von Bildern.
	Diese Technologien finden Anwendung in vielen Bereichen wie Medizin, Finanzen und Industrie.`

	resp, err := client.Summarize(ctx, &babbagepb.SummarizeRequest{
		Text:      longText,
		MaxLength: 50,
	})
	requireNoError(t, err, "Summarize failed")
	requireNotEmpty(t, resp.Summary, "Summary should not be empty")

	t.Logf("Original length: %d chars", len(longText))
	t.Logf("Summary length: %d chars", len(resp.Summary))
	t.Logf("Summary: %s", resp.Summary)
}

func TestBabbage_Classify(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	logTestStart(t, "Babbage", "Classify")

	conn := dialGRPC(t, cfg.BabbageAddr)
	client := babbagepb.NewBabbageServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	resp, err := client.Classify(ctx, &babbagepb.ClassifyRequest{
		Text:       "Der neue iPhone 15 bietet verbesserte Kameraleistung und längere Akkulaufzeit.",
		Categories: []string{"Technologie", "Sport", "Politik", "Wirtschaft"},
	})
	requireNoError(t, err, "Classify failed")

	t.Logf("Primary category: %s (confidence: %.2f)", resp.Category, resp.Confidence)
	if len(resp.Scores) > 0 {
		t.Logf("All scores:")
		for _, s := range resp.Scores {
			t.Logf("  - %s: %.2f", s.Category, s.Score)
		}
	}
}

func TestBabbage_Translate(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama") // LLM needed for translation
	logTestStart(t, "Babbage", "Translate")

	conn := dialGRPC(t, cfg.BabbageAddr)
	client := babbagepb.NewBabbageServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	resp, err := client.Translate(ctx, &babbagepb.TranslateRequest{
		Text:           "Hallo, wie geht es dir?",
		SourceLanguage: "de",
		TargetLanguage: "en",
	})
	requireNoError(t, err, "Translate failed")
	requireNotEmpty(t, resp.TranslatedText, "Translation should not be empty")

	t.Logf("Original (de): Hallo, wie geht es dir?")
	t.Logf("Translated (en): %s", resp.TranslatedText)
	t.Logf("Source: %s, Target: %s", resp.SourceLanguage, resp.TargetLanguage)
}
