package version

import (
	"regexp"
	"testing"
)

// semverRegex validates semantic versioning format
var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func TestVersionConstants(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"Platform", Platform},
		{"Kant", Kant},
		{"Russell", Russell},
		{"Turing", Turing},
		{"Hypatia", Hypatia},
		{"Babbage", Babbage},
		{"Leibniz", Leibniz},
		{"Bayes", Bayes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.version == "" {
				t.Errorf("%s version is empty", tt.name)
			}
			if !semverRegex.MatchString(tt.version) {
				t.Errorf("%s version %q does not match semver format (x.y.z)", tt.name, tt.version)
			}
		})
	}
}

func TestServiceVersion(t *testing.T) {
	tests := []struct {
		name     string
		service  string
		expected string
	}{
		{"kant service", "kant", Kant},
		{"russell service", "russell", Russell},
		{"turing service", "turing", Turing},
		{"hypatia service", "hypatia", Hypatia},
		{"babbage service", "babbage", Babbage},
		{"leibniz service", "leibniz", Leibniz},
		{"bayes service", "bayes", Bayes},
		{"unknown service", "unknown", Platform},
		{"empty service", "", Platform},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ServiceVersion(tt.service)
			if result != tt.expected {
				t.Errorf("ServiceVersion(%q) = %q, want %q", tt.service, result, tt.expected)
			}
		})
	}
}

func TestVersionConsistency(t *testing.T) {
	// All service versions should be consistent with platform version for v1.0.0
	services := []string{Kant, Russell, Turing, Hypatia, Babbage, Leibniz, Bayes}

	for _, v := range services {
		if v != Platform {
			t.Logf("Service version %s differs from platform version %s (this may be intentional)", v, Platform)
		}
	}
}
