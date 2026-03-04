package utils

import (
	"regexp"
	"strings"
)

var (
	invalidChars = regexp.MustCompile(`[^a-z0-9]+`)
)

// Slugify converts a string to all lower case, strips non alphanumeric and hyphen, replaces spaces with hyphens
func Slugify(title string) string {
	// lowercase, replace spaces with hyphens, strip invalid chars
	result := strings.ToLower(title)
	result = invalidChars.ReplaceAllString(result, "-")

	// remove leading and trailing hyphens
	result = strings.Trim(result, "-")

	return result
}
