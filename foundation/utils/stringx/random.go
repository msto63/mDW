// File: random.go
// Title: Secure String Generation Utilities
// Description: Implements secure random string generation for various use cases
//              including passwords, tokens, and identifiers. Uses crypto/rand
//              for cryptographically secure randomness.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with secure random generation

package stringx

import (
	"crypto/rand"
	"math/big"
)

const (
	// Character sets for random string generation
	LettersLowercase = "abcdefghijklmnopqrstuvwxyz"
	LettersUppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Letters          = LettersLowercase + LettersUppercase
	Digits           = "0123456789"
	Alphanumeric     = Letters + Digits
	
	// Safe characters for URLs and filenames (excluding ambiguous characters)
	URLSafe = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	
	// Human-readable characters (excluding visually similar characters like 0, O, l, 1)
	HumanReadable = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	
	// Special characters for password generation
	SpecialChars = "!@#$%^&*()_+-=[]{}|;:,.<>?"
)

// RandomString generates a cryptographically secure random string of the specified length
// using the provided character set. If charset is empty, it defaults to Alphanumeric.
func RandomString(length int, charset string) (string, error) {
	if length <= 0 {
		return "", nil
	}
	
	if charset == "" {
		charset = Alphanumeric
	}
	
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	
	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[randomIndex.Int64()]
	}
	
	return string(result), nil
}

// RandomAlphanumeric generates a random alphanumeric string of the specified length.
// This is a convenience function that uses the Alphanumeric character set.
func RandomAlphanumeric(length int) (string, error) {
	return RandomString(length, Alphanumeric)
}

// RandomHex generates a random hexadecimal string of the specified length.
// The resulting string will contain only characters 0-9 and a-f.
func RandomHex(length int) (string, error) {
	return RandomString(length, "0123456789abcdef")
}

// RandomURLSafe generates a random URL-safe string of the specified length.
// The resulting string is safe to use in URLs and filenames.
func RandomURLSafe(length int) (string, error) {
	return RandomString(length, URLSafe)
}

// RandomHumanReadable generates a random human-readable string of the specified length.
// Excludes visually similar characters to reduce transcription errors.
func RandomHumanReadable(length int) (string, error) {
	return RandomString(length, HumanReadable)
}

// RandomPassword generates a secure random password with the specified length.
// The password will contain a mix of letters, digits, and special characters.
func RandomPassword(length int) (string, error) {
	if length < 4 {
		// Too short for a secure password
		return RandomString(length, Alphanumeric+SpecialChars)
	}
	
	// Ensure the password contains at least one character from each category
	password := make([]byte, length)
	categories := []string{
		LettersLowercase,
		LettersUppercase, 
		Digits,
		SpecialChars,
	}
	
	// Fill first positions with one character from each category
	for i, category := range categories {
		if i >= length {
			break
		}
		char, err := RandomString(1, category)
		if err != nil {
			return "", err
		}
		password[i] = char[0]
	}
	
	// Fill remaining positions with any allowed characters
	remaining := length - len(categories)
	if remaining > 0 {
		remainingChars, err := RandomString(remaining, Alphanumeric+SpecialChars)
		if err != nil {
			return "", err
		}
		copy(password[len(categories):], remainingChars)
	}
	
	// Shuffle the password to avoid predictable patterns
	return shuffleString(string(password))
}

// shuffleString randomly shuffles the characters in a string
func shuffleString(s string) (string, error) {
	runes := []rune(s)
	length := len(runes)
	
	for i := length - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		runes[i], runes[j.Int64()] = runes[j.Int64()], runes[i]
	}
	
	return string(runes), nil
}