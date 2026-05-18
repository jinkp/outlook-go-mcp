package report

import "strings"

// MatchesVIP returns true when from matches any entry in list.
// An entry starting with '@' is a domain suffix match (case-insensitive).
// Any other entry is an exact address match (case-insensitive).
func MatchesVIP(from string, list []string) bool {
	fromLower := strings.ToLower(strings.TrimSpace(from))
	for _, entry := range list {
		entryLower := strings.ToLower(strings.TrimSpace(entry))
		if entryLower == "" {
			continue
		}
		if strings.HasPrefix(entryLower, "@") {
			// Domain suffix match
			if strings.HasSuffix(fromLower, entryLower) {
				return true
			}
		} else {
			// Exact address match
			if fromLower == entryLower {
				return true
			}
		}
	}
	return false
}
