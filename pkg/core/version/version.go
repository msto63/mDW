// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     version
// Description: Central version management for all services
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package version

// Version constants for all mDW services
const (
	// Platform version
	Platform = "1.0.0"

	// Service versions
	Kant       = "1.0.0"
	Russell    = "1.0.0"
	Turing     = "1.0.0"
	Hypatia    = "1.0.0"
	Babbage    = "1.0.0"
	Leibniz    = "1.0.0"
	Bayes      = "1.0.0"
	Platon     = "1.0.0"
	Aristoteles = "1.0.0"
)

// ServiceVersion returns the version for a given service name
func ServiceVersion(name string) string {
	switch name {
	case "kant":
		return Kant
	case "russell":
		return Russell
	case "turing":
		return Turing
	case "hypatia":
		return Hypatia
	case "babbage":
		return Babbage
	case "leibniz":
		return Leibniz
	case "bayes":
		return Bayes
	case "platon":
		return Platon
	case "aristoteles":
		return Aristoteles
	default:
		return Platform
	}
}
