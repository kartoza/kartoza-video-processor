package spellcheck

import (
	"testing"
)

func TestNewSpellChecker(t *testing.T) {
	sc := NewSpellChecker()
	if sc == nil {
		t.Fatal("expected non-nil spell checker")
	}
	if sc.model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestCheckUSSpellings(t *testing.T) {
	sc := NewSpellChecker()

	tests := []struct {
		input    string
		expected string // Expected UK spelling suggestion
	}{
		{"color", "colour"},
		{"Color", "colour"},
		{"organize", "organise"},
		{"center", "centre"},
		{"gray", "grey"},
		{"analyze", "analyse"},
		{"behavior", "behaviour"},
	}

	for _, tt := range tests {
		issues := sc.Check(tt.input)
		found := false
		for _, issue := range issues {
			if issue.Type == "spelling" && len(issue.Suggestions) > 0 {
				if issue.Suggestions[0] == tt.expected {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("expected UK spelling suggestion '%s' for '%s'", tt.expected, tt.input)
		}
	}
}

func TestCheckNoIssuesForCorrectText(t *testing.T) {
	sc := NewSpellChecker()

	// Common correct phrases
	correctTexts := []string{
		"Introduction to QGIS",
		"Learn how to use maps",
		"This is a tutorial",
		"Creating a new layer",
	}

	for _, text := range correctTexts {
		issues := sc.Check(text)
		// Should have no or very few issues
		if len(issues) > 2 {
			t.Errorf("expected few issues for correct text '%s', got %d issues", text, len(issues))
		}
	}
}

func TestCheckGrammar(t *testing.T) {
	sc := NewSpellChecker()

	// Text with double spaces
	issues := sc.Check("This  has  double  spaces")
	foundDoubleSpace := false
	for _, issue := range issues {
		if issue.Type == "grammar" && issue.Message == "Multiple consecutive spaces detected" {
			foundDoubleSpace = true
			break
		}
	}
	if !foundDoubleSpace {
		t.Error("expected to detect double spaces")
	}
}

func TestFormatIssues(t *testing.T) {
	issues := []Issue{
		{Word: "color", Message: "US spelling", Suggestions: []string{"colour"}, Type: "spelling"},
		{Word: "test", Message: "Grammar issue", Suggestions: []string{}, Type: "grammar"},
	}

	formatted := FormatIssues(issues)
	if formatted == "" {
		t.Error("expected non-empty formatted output")
	}
	if len(formatted) < 10 {
		t.Error("formatted output seems too short")
	}
}

func TestFormatIssuesEmpty(t *testing.T) {
	formatted := FormatIssues([]Issue{})
	if formatted != "" {
		t.Error("expected empty string for no issues")
	}
}

func TestExtractWords(t *testing.T) {
	words := extractWords("Hello world test")
	if len(words) != 3 {
		t.Errorf("expected 3 words, got %d", len(words))
	}
	if words[0].text != "Hello" || words[0].position != 0 {
		t.Errorf("unexpected first word: %+v", words[0])
	}
}

func TestIsLikelyAcronym(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"QGIS", true},
		{"API", true},
		{"qgis", false},
		{"Hello", false},
		{"A", false},
		{"VERYLONGACRONYM", false},
	}

	for _, tt := range tests {
		result := isLikelyAcronym(tt.input)
		if result != tt.expected {
			t.Errorf("isLikelyAcronym(%s) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}
