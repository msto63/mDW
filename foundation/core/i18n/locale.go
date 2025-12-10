// File: locale.go
// Title: Locale Detection and Management Implementation  
// Description: Implements locale detection from HTTP Accept-Language headers,
//              browser preferences, and system settings with locale matching
//              and quality score parsing.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of locale detection

package i18n

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	"github.com/msto63/mDW/foundation/utils/stringx"
)

// LocalePreference represents a locale preference with quality score
type LocalePreference struct {
	Locale  string  // Locale code (e.g., "en", "en-US", "de-DE")
	Quality float64 // Quality score (0.0 - 1.0)
}

// DetectLocale detects the best matching locale from Accept-Language header
func (m *Manager) DetectLocale(acceptLanguage string) string {
	if stringx.IsBlank(acceptLanguage) {
		return m.defaultLocale
	}

	// Parse Accept-Language header
	preferences := parseAcceptLanguage(acceptLanguage)
	if len(preferences) == 0 {
		return m.defaultLocale
	}

	// Get available locales
	availableLocales := m.GetAvailableLocales()
	
	// Find best match
	bestMatch := m.findBestLocaleMatch(preferences, availableLocales)
	if bestMatch != "" {
		return bestMatch
	}

	return m.defaultLocale
}

// parseAcceptLanguage parses an Accept-Language header into locale preferences
func parseAcceptLanguage(acceptLang string) []LocalePreference {
	var preferences []LocalePreference

	// Split by comma and process each language tag
	parts := strings.Split(acceptLang, ",")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if stringx.IsBlank(part) {
			continue
		}

		// Parse language tag and quality value
		// Format: "en-US;q=0.9" or "en;q=0.8" or "de"
		var locale string
		var quality float64 = 1.0 // Default quality

		if strings.Contains(part, ";") {
			// Has quality value
			subParts := strings.Split(part, ";")
			locale = strings.TrimSpace(subParts[0])
			
			// Parse quality value
			for _, subPart := range subParts[1:] {
				subPart = strings.TrimSpace(subPart)
				if strings.HasPrefix(subPart, "q=") {
					qValue := strings.TrimPrefix(subPart, "q=")
					if q, err := strconv.ParseFloat(qValue, 64); err == nil {
						quality = q
					}
					break
				}
			}
		} else {
			// No quality value
			locale = part
		}

		if locale != "" {
			preferences = append(preferences, LocalePreference{
				Locale:  locale,
				Quality: quality,
			})
		}
	}

	// Sort by quality (highest first)
	sort.Slice(preferences, func(i, j int) bool {
		return preferences[i].Quality > preferences[j].Quality
	})

	return preferences
}

// findBestLocaleMatch finds the best matching locale from available locales
func (m *Manager) findBestLocaleMatch(preferences []LocalePreference, availableLocales []string) string {
	// Create sets for faster lookup
	availableSet := make(map[string]bool)
	availableBaseLangs := make(map[string]string) // base language -> full locale
	
	for _, locale := range availableLocales {
		availableSet[locale] = true
		
		// Extract base language (e.g., "en-US" -> "en")
		baseLang := strings.Split(locale, "-")[0]
		if _, exists := availableBaseLangs[baseLang]; !exists {
			availableBaseLangs[baseLang] = locale
		}
	}

	// Try to match preferences in order of quality
	for _, pref := range preferences {
		locale := strings.ToLower(pref.Locale)
		
		// 1. Try exact match
		if availableSet[locale] {
			return locale
		}

		// 2. Try case-insensitive exact match
		for _, available := range availableLocales {
			if strings.EqualFold(locale, available) {
				return available
			}
		}

		// 3. Try base language match (e.g., "en-US" matches "en")
		baseLang := strings.Split(locale, "-")[0]
		if fullLocale, exists := availableBaseLangs[baseLang]; exists {
			return fullLocale
		}

		// 4. Try prefix match (e.g., "en" matches "en-US")
		for _, available := range availableLocales {
			if strings.HasPrefix(strings.ToLower(available), baseLang+"-") {
				return available
			}
		}
	}

	return ""
}

// NormalizeLocale normalizes a locale string to standard format
func NormalizeLocale(locale string) string {
	if stringx.IsBlank(locale) {
		return ""
	}

	// Convert to lowercase for processing
	locale = strings.ToLower(locale)
	
	// Handle common separators
	locale = strings.ReplaceAll(locale, "_", "-")
	
	// Split into parts
	parts := strings.Split(locale, "-")
	if len(parts) == 0 {
		return ""
	}

	// Language code (lowercase)
	language := parts[0]
	if len(language) != 2 && len(language) != 3 {
		return ""
	}

	// Country code (uppercase if present)
	if len(parts) > 1 && len(parts[1]) == 2 {
		country := strings.ToUpper(parts[1])
		return language + "-" + country
	}

	return language
}

// ValidateLocale validates if a locale string is in valid format
func ValidateLocale(locale string) error {
	if stringx.IsBlank(locale) {
		return mdwerror.New("locale cannot be empty").WithCode(mdwerror.CodeValidationFailed).WithOperation("i18n.ValidateLocale")
	}

	normalized := NormalizeLocale(locale)
	if stringx.IsBlank(normalized) {
		return mdwerror.New("invalid locale format").WithCode(mdwerror.CodeValidationFailed).WithOperation("i18n.ValidateLocale").WithDetail("locale", locale).WithDetail("expected_format", "e.g., 'en', 'en-US'")
	}

	return nil
}

// SplitLocale splits a locale into language and country parts
func SplitLocale(locale string) (language, country string) {
	normalized := NormalizeLocale(locale)
	if stringx.IsBlank(normalized) {
		return "", ""
	}

	parts := strings.Split(normalized, "-")
	language = parts[0]
	
	if len(parts) > 1 {
		country = parts[1]
	}

	return language, country
}

// GetLocaleDisplayName returns a human-readable display name for a locale
func GetLocaleDisplayName(locale string) string {
	// Map of common locales to display names
	displayNames := map[string]string{
		"en":    "English",
		"en-US": "English (United States)",
		"en-GB": "English (United Kingdom)", 
		"en-CA": "English (Canada)",
		"en-AU": "English (Australia)",
		"de":    "Deutsch",
		"de-DE": "Deutsch (Deutschland)",
		"de-AT": "Deutsch (Österreich)",
		"de-CH": "Deutsch (Schweiz)",
		"fr":    "Français",
		"fr-FR": "Français (France)",
		"fr-CA": "Français (Canada)",
		"fr-BE": "Français (Belgique)",
		"es":    "Español",
		"es-ES": "Español (España)",
		"es-MX": "Español (México)",
		"es-AR": "Español (Argentina)",
		"it":    "Italiano",
		"it-IT": "Italiano (Italia)",
		"pt":    "Português",
		"pt-BR": "Português (Brasil)",
		"pt-PT": "Português (Portugal)",
		"ru":    "Русский",
		"ru-RU": "Русский (Россия)",
		"zh":    "中文",
		"zh-CN": "中文 (简体)",
		"zh-TW": "中文 (繁體)",
		"ja":    "日本語",
		"ja-JP": "日本語 (日本)",
		"ko":    "한국어",
		"ko-KR": "한국어 (대한민국)",
		"ar":    "العربية",
		"ar-SA": "العربية (المملكة العربية السعودية)",
		"hi":    "हिन्दी",
		"hi-IN": "हिन्दी (भारत)",
		"tr":    "Türkçe",
		"tr-TR": "Türkçe (Türkiye)",
		"pl":    "Polski",
		"pl-PL": "Polski (Polska)",
		"nl":    "Nederlands",
		"nl-NL": "Nederlands (Nederland)",
		"nl-BE": "Nederlands (België)",
		"sv":    "Svenska",
		"sv-SE": "Svenska (Sverige)",
		"da":    "Dansk",
		"da-DK": "Dansk (Danmark)",
		"no":    "Norsk",
		"no-NO": "Norsk (Norge)",
		"fi":    "Suomi",
		"fi-FI": "Suomi (Suomi)",
	}

	normalized := NormalizeLocale(locale)
	if displayName, exists := displayNames[normalized]; exists {
		return displayName
	}

	// Fallback to the locale code itself
	if normalized != "" {
		return normalized
	}
	return locale
}

// GetSupportedLocales returns a list of commonly supported locales
func GetSupportedLocales() []string {
	return []string{
		"en", "en-US", "en-GB", "en-CA", "en-AU",
		"de", "de-DE", "de-AT", "de-CH",
		"fr", "fr-FR", "fr-CA", "fr-BE",
		"es", "es-ES", "es-MX", "es-AR",
		"it", "it-IT",
		"pt", "pt-BR", "pt-PT",
		"ru", "ru-RU",
		"zh", "zh-CN", "zh-TW",
		"ja", "ja-JP",
		"ko", "ko-KR",
		"ar", "ar-SA",
		"hi", "hi-IN",
		"tr", "tr-TR",
		"pl", "pl-PL",
		"nl", "nl-NL", "nl-BE",
		"sv", "sv-SE",
		"da", "da-DK",
		"no", "no-NO",
		"fi", "fi-FI",
	}
}

// FormatLocaleForFilename formats a locale for use in filenames
func FormatLocaleForFilename(locale string) string {
	normalized := NormalizeLocale(locale)
	// Replace hyphens with underscores for filename compatibility
	return strings.ReplaceAll(normalized, "-", "_")
}

// ParseLocaleFromFilename extracts locale from a filename
func ParseLocaleFromFilename(filename string) string {
	// Remove file extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Convert underscores back to hyphens
	locale := strings.ReplaceAll(name, "_", "-")
	
	return NormalizeLocale(locale)
}