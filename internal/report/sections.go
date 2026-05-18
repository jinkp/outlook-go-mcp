package report

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/jinkp/outlook-go-mcp/internal/domain"
)

// buildUnanswered returns emails that have no matching reply in Sent Items.
// For each email, it searches Sent Items for a reply matching the normalized subject.
// An email is unanswered when SearchEmails returns 0 results.
func buildUnanswered(ctx context.Context, emails []domain.Email, since time.Time, mail domain.MailStore, max int) []domain.Email {
	result := make([]domain.Email, 0)
	for _, email := range emails {
		if len(result) >= max {
			break
		}
		subject := NormalizeSubject(email.Subject)
		replies, err := mail.SearchEmails(ctx, domain.SearchEmailsParams{
			Query:      subject,
			Folder:     "Sent",
			Since:      since,
			MaxResults: 1,
		})
		if err != nil || len(replies) == 0 {
			result = append(result, email)
		}
	}
	return result
}

// buildVIP returns emails whose From field matches any entry in vipList.
func buildVIP(emails []domain.Email, vipList []string, max int) []domain.Email {
	if len(vipList) == 0 {
		return nil
	}
	result := make([]domain.Email, 0)
	for _, email := range emails {
		if len(result) >= max {
			break
		}
		if MatchesVIP(email.From, vipList) {
			result = append(result, email)
		}
	}
	return result
}

// buildAttachments returns emails where HasAttachments is true.
func buildAttachments(emails []domain.Email, max int) []domain.Email {
	result := make([]domain.Email, 0)
	for _, email := range emails {
		if len(result) >= max {
			break
		}
		if email.HasAttachments {
			result = append(result, email)
		}
	}
	return result
}

// buildThreads groups emails by NormalizeSubject and returns groups with at least minSize messages.
// Groups are sorted by count descending and capped at max.
func buildThreads(emails []domain.Email, max int, minSize int) []ThreadGroup {
	groups := make(map[string][]domain.Email)
	for _, email := range emails {
		normalized := NormalizeSubject(email.Subject)
		groups[normalized] = append(groups[normalized], email)
	}

	result := make([]ThreadGroup, 0)
	for normalized, group := range groups {
		if len(group) < minSize {
			continue
		}

		// Find earliest sender and latest time
		earliestSender := ""
		var latestTime time.Time
		var earliestTime time.Time

		for i, email := range group {
			if i == 0 || email.Date.Before(earliestTime) {
				earliestTime = email.Date
				earliestSender = email.From
			}
			if email.Date.After(latestTime) {
				latestTime = email.Date
			}
		}

		result = append(result, ThreadGroup{
			NormalizedSubject: normalized,
			Count:             len(group),
			EarliestSender:    earliestSender,
			LatestTime:        latestTime,
			Emails:            group,
		})
	}

	// Sort by count descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	if len(result) > max {
		result = result[:max]
	}
	return result
}

// buildCalendar pairs calendar events with a count of related emails.
// An email is related when any word (>4 chars) from the event title appears in the email Subject (case-insensitive).
func buildCalendar(events []domain.CalendarEvent, emails []domain.Email, max int) []CalendarEventWithHint {
	result := make([]CalendarEventWithHint, 0, len(events))
	for i, event := range events {
		if i >= max {
			break
		}
		keywords := extractKeywords(event.Title)
		count := 0
		for _, email := range emails {
			subjectLower := strings.ToLower(email.Subject)
			for _, kw := range keywords {
				if strings.Contains(subjectLower, kw) {
					count++
					break
				}
			}
		}
		result = append(result, CalendarEventWithHint{
			Event:         event,
			RelatedEmails: count,
		})
	}
	return result
}

// extractKeywords returns all words with more than 4 characters from s, lowercased.
func extractKeywords(s string) []string {
	words := strings.Fields(s)
	result := make([]string, 0, len(words))
	for _, word := range words {
		clean := strings.ToLower(strings.Trim(word, ".,;:!?\"'"))
		if len(clean) > 4 {
			result = append(result, clean)
		}
	}
	return result
}
