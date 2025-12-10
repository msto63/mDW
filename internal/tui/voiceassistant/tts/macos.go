// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     tts
// Description: macOS native TTS using 'say' command
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package tts

import (
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// MacOSSay implements text-to-speech using macOS say command
type MacOSSay struct {
	voice string
	rate  int
}

// NewMacOSSay creates a new macOS say TTS synthesizer
func NewMacOSSay(voice string) *MacOSSay {
	return &MacOSSay{
		voice: voice,
		rate:  200, // Words per minute
	}
}

// NewMacOSSayWithRate creates a new macOS say TTS synthesizer with custom rate
func NewMacOSSayWithRate(voice string, rate int) *MacOSSay {
	if rate <= 0 {
		rate = 200
	}
	return &MacOSSay{
		voice: voice,
		rate:  rate,
	}
}

// IsAvailable checks if macOS say is available
func (m *MacOSSay) IsAvailable() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := exec.LookPath("say")
	return err == nil
}

// Synthesize is not supported for say (it speaks directly)
func (m *MacOSSay) Synthesize(ctx context.Context, text string) ([]byte, error) {
	// macOS say speaks directly, doesn't return audio data
	return nil, nil
}

// Speak speaks the text directly using macOS say
func (m *MacOSSay) Speak(ctx context.Context, text string) error {
	args := []string{}

	if m.voice != "" {
		args = append(args, "-v", m.voice)
	}

	// Use configured rate
	if m.rate > 0 {
		args = append(args, "-r", strconv.Itoa(m.rate))
	}

	args = append(args, text)

	cmd := exec.CommandContext(ctx, "say", args...)
	return cmd.Run()
}

// SpeakStreaming speaks text sentence by sentence for faster feedback
func (m *MacOSSay) SpeakStreaming(ctx context.Context, text string) error {
	// Split into sentences
	sentences := splitIntoSentences(text)

	for _, sentence := range sentences {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		args := []string{}
		if m.voice != "" {
			args = append(args, "-v", m.voice)
		}
		if m.rate > 0 {
			args = append(args, "-r", strconv.Itoa(m.rate))
		}
		args = append(args, sentence)

		cmd := exec.CommandContext(ctx, "say", args...)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

// splitIntoSentences splits text into sentences
func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' || r == ':' || r == '\n' {
			s := current.String()
			if len(s) > 1 {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	// Add remaining text
	if current.Len() > 0 {
		sentences = append(sentences, current.String())
	}

	return sentences
}

// SampleRate returns 0 as say doesn't expose this
func (m *MacOSSay) SampleRate() int {
	return 0
}

// Close is a no-op
func (m *MacOSSay) Close() error {
	return nil
}
