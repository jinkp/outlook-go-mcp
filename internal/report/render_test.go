package report

import (
	"strings"
	"testing"
	"time"

	"github.com/jinkp/outlook-go-mcp/internal/domain"
)

func TestRenderMarkdownContainsAllSections(t *testing.T) {
	rpt := &Report{
		GeneratedAt: time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC),
		Unanswered: []domain.Email{
			{ID: "1", Subject: "Budget Review", From: "alice@example.com", Date: time.Now()},
		},
		VIPEmails: []domain.Email{
			{ID: "2", Subject: "VIP message", From: "boss@vip.com", Date: time.Now()},
		},
		Attachments: []domain.Email{
			{ID: "3", Subject: "Report attached", From: "hr@example.com", HasAttachments: true, Date: time.Now()},
		},
		Threads: []ThreadGroup{
			{NormalizedSubject: "BUDGET", Count: 5, EarliestSender: "alice@example.com", LatestTime: time.Now()},
		},
		Events: []CalendarEventWithHint{
			{
				Event:         domain.CalendarEvent{ID: "evt1", Title: "Budget Meeting", Start: time.Date(2026, time.May, 18, 9, 0, 0, 0, time.UTC)},
				RelatedEmails: 2,
			},
		},
	}

	output := RenderMarkdown(rpt)

	requiredHeaders := []string{
		"## Unanswered Emails",
		"## VIP Emails",
		"## Attachments",
		"## Active Threads",
		"## Calendar",
	}
	for _, header := range requiredHeaders {
		if !strings.Contains(output, header) {
			t.Errorf("RenderMarkdown() output missing header %q", header)
		}
	}

	// Verify thread group format
	if !strings.Contains(output, "**BUDGET** (5 messages)") {
		t.Errorf("RenderMarkdown() output missing thread group format; output:\n%s", output)
	}

	// Verify calendar hint
	if !strings.Contains(output, "related emails: 2") {
		t.Errorf("RenderMarkdown() output missing calendar hint; output:\n%s", output)
	}
}

func TestRenderMarkdownEmptySectionsShowNoItems(t *testing.T) {
	rpt := &Report{
		GeneratedAt: time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC),
		Unanswered:  nil,
		VIPEmails:   nil,
		Attachments: nil,
		Threads:     nil,
		Events:      nil,
	}

	output := RenderMarkdown(rpt)

	if !strings.Contains(output, "No items found.") {
		t.Errorf("RenderMarkdown() empty report should contain 'No items found.'; output:\n%s", output)
	}

	// All sections should still be present
	requiredHeaders := []string{
		"## Unanswered Emails",
		"## VIP Emails",
		"## Attachments",
		"## Active Threads",
		"## Calendar",
	}
	for _, header := range requiredHeaders {
		if !strings.Contains(output, header) {
			t.Errorf("RenderMarkdown() output missing header %q even for empty report", header)
		}
	}
}

func TestRenderMarkdownCalendarHintOnlyShownWhenPositive(t *testing.T) {
	rpt := &Report{
		GeneratedAt: time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC),
		Events: []CalendarEventWithHint{
			{
				Event:         domain.CalendarEvent{ID: "evt1", Title: "Standup", Start: time.Date(2026, time.May, 18, 9, 0, 0, 0, time.UTC)},
				RelatedEmails: 0,
			},
		},
	}

	output := RenderMarkdown(rpt)

	// When RelatedEmails == 0, the hint should NOT be shown
	if strings.Contains(output, "related emails:") {
		t.Errorf("RenderMarkdown() should NOT show calendar hint when RelatedEmails=0; output:\n%s", output)
	}
}
