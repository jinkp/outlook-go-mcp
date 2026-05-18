package report

import (
	"fmt"
	"strings"
)

// RenderMarkdown produces a Markdown report from r with all five sections.
func RenderMarkdown(r *Report) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Daily Email Report\n")
	fmt.Fprintf(&sb, "_Generated: %s_\n\n", r.GeneratedAt.Format("2006-01-02 15:04 MST"))

	// Section 1: Unanswered Emails
	sb.WriteString("## Unanswered Emails\n\n")
	if len(r.Unanswered) == 0 {
		sb.WriteString("No items found.\n")
	} else {
		for _, email := range r.Unanswered {
			fmt.Fprintf(&sb, "- **%s** — from %s (%s)\n", email.Subject, email.From, email.Date.Format("15:04"))
		}
	}
	sb.WriteString("\n")

	// Section 2: VIP Emails
	sb.WriteString("## VIP Emails\n\n")
	if len(r.VIPEmails) == 0 {
		sb.WriteString("No items found.\n")
	} else {
		for _, email := range r.VIPEmails {
			fmt.Fprintf(&sb, "- **%s** — from %s (%s)\n", email.Subject, email.From, email.Date.Format("15:04"))
		}
	}
	sb.WriteString("\n")

	// Section 3: Attachments
	sb.WriteString("## Attachments\n\n")
	if len(r.Attachments) == 0 {
		sb.WriteString("No items found.\n")
	} else {
		for _, email := range r.Attachments {
			fmt.Fprintf(&sb, "- **%s** — from %s (%s)\n", email.Subject, email.From, email.Date.Format("15:04"))
		}
	}
	sb.WriteString("\n")

	// Section 4: Active Threads
	sb.WriteString("## Active Threads\n\n")
	if len(r.Threads) == 0 {
		sb.WriteString("No items found.\n")
	} else {
		for _, thread := range r.Threads {
			fmt.Fprintf(&sb, "- **%s** (%d messages) — last: %s, started by: %s\n",
				thread.NormalizedSubject,
				thread.Count,
				thread.LatestTime.Format("15:04"),
				thread.EarliestSender,
			)
		}
	}
	sb.WriteString("\n")

	// Section 5: Calendar
	sb.WriteString("## Calendar\n\n")
	if len(r.Events) == 0 {
		sb.WriteString("No items found.\n")
	} else {
		for _, evtHint := range r.Events {
			if evtHint.RelatedEmails > 0 {
				fmt.Fprintf(&sb, "- %s %s [related emails: %d]\n",
					evtHint.Event.Start.Format("15:04"),
					evtHint.Event.Title,
					evtHint.RelatedEmails,
				)
			} else {
				fmt.Fprintf(&sb, "- %s %s\n",
					evtHint.Event.Start.Format("15:04"),
					evtHint.Event.Title,
				)
			}
		}
	}
	sb.WriteString("\n")

	return sb.String()
}
