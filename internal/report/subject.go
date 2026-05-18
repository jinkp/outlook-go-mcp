package report

import "strings"

// prefixWords is the list of prefix keywords to strip (without the colon), case-insensitive.
var prefixWords = []string{
	"REENVIAR",
	"REENV",
	"FWD",
	"RES",
	"RE",
	"FW",
	"RV",
	"AW",
	"WG",
	"TR",
}

// NormalizeSubject strips all reply/forward prefixes from s iteratively and returns
// the result in upper case. If the result is empty, returns "(NO SUBJECT)".
// A prefix matches when the uppercased string starts with WORD: followed by optional whitespace.
func NormalizeSubject(s string) string {
	s = strings.TrimSpace(s)

	for {
		upper := strings.ToUpper(s)
		stripped := false
		for _, word := range prefixWords {
			candidate := word + ":"
			if strings.HasPrefix(upper, candidate) {
				s = strings.TrimSpace(s[len(candidate):])
				stripped = true
				break
			}
		}
		if !stripped {
			break
		}
	}

	result := strings.TrimSpace(strings.ToUpper(s))
	if result == "" {
		return "(NO SUBJECT)"
	}
	return result
}
