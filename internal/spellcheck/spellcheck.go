package spellcheck

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/sajari/fuzzy"
)

// SpellChecker provides spell checking functionality with UK English
type SpellChecker struct {
	model *fuzzy.Model
}

// Issue represents a spelling or grammar issue
type Issue struct {
	Word        string   // The problematic word
	Position    int      // Position in the text (character index)
	Suggestions []string // Suggested corrections
	Type        string   // "spelling" or "grammar"
	Message     string   // Human-readable message
}

// commonWords is a basic dictionary of common English words
// This is a minimal set - in production, you'd load a full dictionary
var commonWords = []string{
	// Common words
	"the", "be", "to", "of", "and", "a", "in", "that", "have", "i",
	"it", "for", "not", "on", "with", "he", "as", "you", "do", "at",
	"this", "but", "his", "by", "from", "they", "we", "say", "her", "she",
	"or", "an", "will", "my", "one", "all", "would", "there", "their", "what",
	"so", "up", "out", "if", "about", "who", "get", "which", "go", "me",
	"when", "make", "can", "like", "time", "no", "just", "him", "know", "take",
	"people", "into", "year", "your", "good", "some", "could", "them", "see", "other",
	"than", "then", "now", "look", "only", "come", "its", "over", "think", "also",
	"back", "after", "use", "two", "how", "our", "work", "first", "well", "way",
	"even", "new", "want", "because", "any", "these", "give", "day", "most", "us",

	// Technical/video related words
	"video", "audio", "recording", "screen", "tutorial", "introduction", "episode",
	"chapter", "part", "series", "guide", "demo", "demonstration", "walkthrough",
	"overview", "review", "update", "feature", "features", "tip", "tips", "trick",
	"tricks", "howto", "how", "learn", "learning", "beginner", "beginners", "advanced",
	"intermediate", "basic", "basics", "complete", "full", "quick", "start", "getting",
	"started", "setup", "install", "installation", "configure", "configuration",

	// GIS/QGIS related words
	"qgis", "gis", "geographic", "information", "system", "systems", "map", "maps",
	"mapping", "layer", "layers", "data", "dataset", "datasets", "spatial", "analysis",
	"raster", "vector", "polygon", "polygons", "point", "points", "line", "lines",
	"coordinate", "coordinates", "projection", "projections", "crs", "georeferencing",
	"digitizing", "digitising", "sketcher", "sketches", "sketching", "plugin", "plugins",
	"processing", "toolbox", "tool", "tools", "attribute", "attributes", "table",
	"symbology", "style", "styles", "styling", "label", "labels", "labeling", "labelling",
	"cartography", "cartographic", "export", "exporting", "import", "importing",
	"database", "databases", "postgis", "postgres", "postgresql", "shapefile", "geojson",
	"wms", "wfs", "wcs", "ogc", "open", "source", "foss", "free", "software",

	// UK English specific spellings
	"colour", "colours", "favour", "favours", "behaviour", "behaviours",
	"honour", "honours", "labour", "labours", "neighbour", "neighbours",
	"analyse", "analysing", "organised", "organising", "organisation",
	"realise", "realising", "recognise", "recognising", "specialise", "specialising",
	"customise", "customising", "visualise", "visualising", "optimise", "optimising",
	"centre", "centres", "metre", "metres", "litre", "litres",
	"theatre", "theatres", "fibre", "fibres",
	"catalogue", "catalogues", "dialogue", "dialogues",
	"programme", "programmes", "practise", "practising",
	"defence", "licence", "offence", "pretence",
	"travelled", "travelling", "traveller", "labelled", "labelling",
	"modelled", "modelling", "cancelled", "cancelling",
	"grey", "cheque", "cheques", "jewellery", "maths",

	// Common technical terms
	"workflow", "workflows", "interface", "interfaces", "api", "apis",
	"button", "buttons", "menu", "menus", "panel", "panels", "window", "windows",
	"click", "clicking", "select", "selecting", "drag", "dragging", "drop", "dropping",
	"zoom", "zooming", "pan", "panning", "scroll", "scrolling",
	"file", "files", "folder", "folders", "directory", "directories",
	"save", "saving", "load", "loading", "create", "creating", "delete", "deleting",
	"edit", "editing", "modify", "modifying", "change", "changing", "update", "updating",
	"add", "adding", "remove", "removing", "insert", "inserting",
	"copy", "copying", "paste", "pasting", "cut", "cutting",
	"undo", "redo", "reset", "clear", "refresh",

	// Numbers as words
	"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
	"eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen",
	"eighteen", "nineteen", "twenty", "thirty", "forty", "fifty", "sixty", "seventy",
	"eighty", "ninety", "hundred", "thousand", "million",

	// Time-related
	"second", "seconds", "minute", "minutes", "hour", "hours", "day", "days",
	"week", "weeks", "month", "months", "year", "years",
	"today", "tomorrow", "yesterday", "morning", "afternoon", "evening", "night",

	// Descriptive words
	"new", "old", "big", "small", "large", "little", "great", "good", "bad",
	"best", "better", "worse", "worst", "high", "low", "long", "short",
	"easy", "hard", "difficult", "simple", "complex", "fast", "slow", "quick",
	"beautiful", "wonderful", "amazing", "awesome", "fantastic", "excellent",
	"important", "essential", "necessary", "useful", "helpful", "powerful",

	// Verbs
	"is", "are", "was", "were", "been", "being", "has", "had", "having",
	"does", "did", "doing", "done", "makes", "made", "making",
	"shows", "showed", "showing", "shown", "explains", "explained", "explaining",
	"demonstrates", "demonstrated", "demonstrating", "teaches", "taught", "teaching",
	"uses", "used", "using", "works", "worked", "working",
	"helps", "helped", "helping", "allows", "allowed", "allowing",
	"includes", "included", "including", "provides", "provided", "providing",
	"requires", "required", "requiring", "needs", "needed", "needing",
	"contains", "contained", "containing", "covers", "covered", "covering",
}

// usToUkSpelling maps US spellings to UK spellings
var usToUkSpelling = map[string]string{
	"color":        "colour",
	"colors":       "colours",
	"favor":        "favour",
	"favors":       "favours",
	"behavior":     "behaviour",
	"behaviors":    "behaviours",
	"honor":        "honour",
	"honors":       "honours",
	"labor":        "labour",
	"labors":       "labours",
	"neighbor":     "neighbour",
	"neighbors":    "neighbours",
	"analyze":      "analyse",
	"analyzing":    "analysing",
	"organize":     "organise",
	"organizing":   "organising",
	"organization": "organisation",
	"realize":      "realise",
	"realizing":    "realising",
	"recognize":    "recognise",
	"recognizing":  "recognising",
	"specialize":   "specialise",
	"specializing": "specialising",
	"customize":    "customise",
	"customizing":  "customising",
	"visualize":    "visualise",
	"visualizing":  "visualising",
	"optimize":     "optimise",
	"optimizing":   "optimising",
	"center":       "centre",
	"centers":      "centres",
	"meter":        "metre",
	"meters":       "metres",
	"liter":        "litre",
	"liters":       "litres",
	"theater":      "theatre",
	"theaters":     "theatres",
	"fiber":        "fibre",
	"fibers":       "fibres",
	"catalog":      "catalogue",
	"catalogs":     "catalogues",
	"dialog":       "dialogue",
	"dialogs":      "dialogues",
	"program":      "programme",
	"programs":     "programmes",
	"practice":     "practise", // verb form
	"practicing":   "practising",
	"defense":      "defence",
	"offense":      "offence",
	"license":      "licence", // noun form
	"gray":         "grey",
	"check":        "cheque", // for banking context
	"jewelry":      "jewellery",
	"math":         "maths",
	"traveled":     "travelled",
	"traveling":    "travelling",
	"traveler":     "traveller",
	"labeled":      "labelled",
	"labeling":     "labelling",
	"modeled":      "modelled",
	"modeling":     "modelling",
	"canceled":     "cancelled",
	"canceling":    "cancelling",
}

// grammarPatterns contains common grammar issues to check
var grammarPatterns = []struct {
	pattern     *regexp.Regexp
	message     string
	suggestion  string
	issueType   string
}{
	{
		pattern:    regexp.MustCompile(`(?i)\ba\s+[aeiou]`),
		message:    "Consider using 'an' before words starting with a vowel sound",
		suggestion: "an",
		issueType:  "grammar",
	},
	{
		pattern:    regexp.MustCompile(`(?i)\ban\s+[bcdfghjklmnpqrstvwxyz]`),
		message:    "Consider using 'a' before words starting with a consonant sound",
		suggestion: "a",
		issueType:  "grammar",
	},
	{
		pattern:    regexp.MustCompile(`(?i)\b(its|it's)\b`),
		message:    "Check: 'its' (possessive) vs 'it's' (it is)",
		suggestion: "",
		issueType:  "grammar",
	},
	{
		pattern:    regexp.MustCompile(`(?i)\b(their|there|they're)\b`),
		message:    "Check: 'their' (possessive), 'there' (place), 'they're' (they are)",
		suggestion: "",
		issueType:  "grammar",
	},
	{
		pattern:    regexp.MustCompile(`(?i)\b(your|you're)\b`),
		message:    "Check: 'your' (possessive) vs 'you're' (you are)",
		suggestion: "",
		issueType:  "grammar",
	},
	{
		pattern:    regexp.MustCompile(`(?i)\s{2,}`),
		message:    "Multiple consecutive spaces detected",
		suggestion: " ",
		issueType:  "grammar",
	},
}

// NewSpellChecker creates a new spell checker with UK English dictionary
func NewSpellChecker() *SpellChecker {
	model := fuzzy.NewModel()
	model.SetThreshold(1) // Only exact matches or 1 edit distance
	model.SetDepth(2)

	// Train the model with our dictionary
	model.Train(commonWords)

	return &SpellChecker{
		model: model,
	}
}

// Check checks the given text for spelling and grammar issues
func (sc *SpellChecker) Check(text string) []Issue {
	var issues []Issue

	// Check for US spellings that should be UK
	issues = append(issues, sc.checkUSSpellings(text)...)

	// Check for spelling errors
	issues = append(issues, sc.checkSpelling(text)...)

	// Check for grammar issues
	issues = append(issues, sc.checkGrammar(text)...)

	return issues
}

// checkUSSpellings checks for US spellings that should be UK
func (sc *SpellChecker) checkUSSpellings(text string) []Issue {
	var issues []Issue
	words := extractWords(text)

	for _, word := range words {
		lower := strings.ToLower(word.text)
		if ukSpelling, exists := usToUkSpelling[lower]; exists {
			issues = append(issues, Issue{
				Word:        word.text,
				Position:    word.position,
				Suggestions: []string{ukSpelling},
				Type:        "spelling",
				Message:     "US spelling detected. UK spelling: " + ukSpelling,
			})
		}
	}

	return issues
}

// checkSpelling checks for spelling errors
func (sc *SpellChecker) checkSpelling(text string) []Issue {
	var issues []Issue
	words := extractWords(text)

	for _, word := range words {
		lower := strings.ToLower(word.text)

		// Skip very short words, numbers, and already flagged US spellings
		if len(lower) < 3 || isNumber(lower) || isLikelyAcronym(word.text) {
			continue
		}

		// Skip if it's a US spelling (already handled)
		if _, isUS := usToUkSpelling[lower]; isUS {
			continue
		}

		// Skip if it's in our dictionary
		if sc.isKnownWord(lower) {
			continue
		}

		// Get suggestions
		suggestions := sc.model.Suggestions(lower, false)
		if len(suggestions) > 0 {
			// Limit suggestions
			if len(suggestions) > 3 {
				suggestions = suggestions[:3]
			}
			issues = append(issues, Issue{
				Word:        word.text,
				Position:    word.position,
				Suggestions: suggestions,
				Type:        "spelling",
				Message:     "Possible spelling error",
			})
		}
	}

	return issues
}

// checkGrammar checks for common grammar issues
func (sc *SpellChecker) checkGrammar(text string) []Issue {
	var issues []Issue

	for _, gp := range grammarPatterns {
		matches := gp.pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			word := text[match[0]:match[1]]
			issues = append(issues, Issue{
				Word:        strings.TrimSpace(word),
				Position:    match[0],
				Suggestions: []string{gp.suggestion},
				Type:        gp.issueType,
				Message:     gp.message,
			})
		}
	}

	return issues
}

// isKnownWord checks if a word is in our dictionary
func (sc *SpellChecker) isKnownWord(word string) bool {
	for _, w := range commonWords {
		if w == word {
			return true
		}
	}
	return false
}

// wordInfo holds information about a word and its position
type wordInfo struct {
	text     string
	position int
}

// extractWords extracts words and their positions from text
func extractWords(text string) []wordInfo {
	var words []wordInfo
	var currentWord strings.Builder
	wordStart := 0

	for i, r := range text {
		if unicode.IsLetter(r) || r == '\'' {
			if currentWord.Len() == 0 {
				wordStart = i
			}
			currentWord.WriteRune(r)
		} else {
			if currentWord.Len() > 0 {
				words = append(words, wordInfo{
					text:     currentWord.String(),
					position: wordStart,
				})
				currentWord.Reset()
			}
		}
	}

	// Don't forget the last word
	if currentWord.Len() > 0 {
		words = append(words, wordInfo{
			text:     currentWord.String(),
			position: wordStart,
		})
	}

	return words
}

// isNumber checks if a string is a number
func isNumber(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}

// isLikelyAcronym checks if a word is likely an acronym (all caps)
func isLikelyAcronym(s string) bool {
	if len(s) < 2 || len(s) > 6 {
		return false
	}
	for _, r := range s {
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// FormatIssues formats issues for display
func FormatIssues(issues []Issue) string {
	if len(issues) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, issue := range issues {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("• ")
		sb.WriteString(issue.Word)
		sb.WriteString(": ")
		sb.WriteString(issue.Message)
		if len(issue.Suggestions) > 0 && issue.Suggestions[0] != "" {
			sb.WriteString(" → ")
			sb.WriteString(strings.Join(issue.Suggestions, ", "))
		}
	}
	return sb.String()
}
