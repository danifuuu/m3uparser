package entry

import (
	"regexp"
	"strings"
)

// ProcessValue applies replace, header-scrub, and term-removal operations to a value.
// This replicates the Python process_value function.
func ProcessValue(value string, replace map[string]string, removeHeader []string) string {
	// Apply replacements.
	for key, replacement := range replace {
		pattern := regexp.MustCompile(regexp.QuoteMeta(key))
		value = pattern.ReplaceAllString(value, replacement)
	}

	// Remove header prefixes: match from the beginning of the string,
	// strip everything up to and including the header term plus trailing spaces/commas.
	for _, header := range removeHeader {
		pattern := regexp.MustCompile(`(?i)^.*?` + regexp.QuoteMeta(header) + `[\s,]*`)
		if loc := pattern.FindStringIndex(value); loc != nil {
			value = strings.TrimSpace(value[loc[1]:])
		}
	}

	return strings.TrimSpace(value)
}

// RemoveAllTerms removes all given terms (and trailing non-space chars) from the value.
func RemoveAllTerms(value string, termSets ...[]string) string {
	for _, terms := range termSets {
		for _, term := range terms {
			pattern := regexp.MustCompile(`\s*` + regexp.QuoteMeta(term) + `\S*`)
			value = pattern.ReplaceAllString(value, "")
		}
	}
	return strings.TrimSpace(value)
}

// CheckExcluded returns true if the group-title matches any exclude pattern (case-insensitive).
func CheckExcluded(groupTitle string, excludeTerms []string) bool {
	for _, term := range excludeTerms {
		re, err := regexp.Compile("(?i)" + term)
		if err != nil {
			continue
		}
		if re.MatchString(groupTitle) {
			return true
		}
	}
	return false
}
