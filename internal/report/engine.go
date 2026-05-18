package report

import (
	"context"
	"log/slog"
	"time"

	"github.com/jinkp/outlook-go-mcp/internal/config"
	"github.com/jinkp/outlook-go-mcp/internal/domain"
)

// Engine orchestrates the generation of a daily email report.
type Engine struct {
	Mail     domain.MailStore
	Calendar domain.CalendarStore
	Config   config.ReportConfig
	Logger   *slog.Logger
	Now      func() time.Time
}

// Report holds all five report sections plus metadata.
type Report struct {
	GeneratedAt time.Time
	Unanswered  []domain.Email
	VIPEmails   []domain.Email
	Attachments []domain.Email
	Threads     []ThreadGroup
	Events      []CalendarEventWithHint
}

// ThreadGroup represents a group of emails sharing the same normalized subject.
type ThreadGroup struct {
	NormalizedSubject string
	Count             int
	EarliestSender    string
	LatestTime        time.Time
	Emails            []domain.Email
}

// CalendarEventWithHint pairs a calendar event with a related-email count.
type CalendarEventWithHint struct {
	Event         domain.CalendarEvent
	RelatedEmails int
}

// NewEngine creates an Engine with the provided dependencies.
// Pass nil for logger to disable logging. Pass nil for now to use time.Now.
func NewEngine(mail domain.MailStore, calendar domain.CalendarStore, cfg config.ReportConfig, logger *slog.Logger, now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{
		Mail:     mail,
		Calendar: calendar,
		Config:   cfg,
		Logger:   logger,
		Now:      now,
	}
}

// Run executes all five report sections and returns a populated Report.
func (e *Engine) Run(ctx context.Context) (*Report, error) {
	now := e.Now()
	since := now.Add(-time.Duration(e.Config.SinceHours) * time.Hour)
	maxResults := e.Config.MaxPerSection
	if maxResults <= 0 {
		maxResults = 20
	}

	// Fetch all emails in the time window
	emails, err := e.Mail.ListEmailsInRange(ctx, domain.ListEmailsInRangeParams{
		Since:      since,
		Until:      now,
		MaxResults: maxResults * 10, // Fetch more to allow filtering
	})
	if err != nil {
		return nil, err
	}

	rpt := &Report{
		GeneratedAt: now,
	}

	rpt.Unanswered = buildUnanswered(ctx, emails, since, e.Mail, maxResults)
	rpt.VIPEmails = buildVIP(emails, e.Config.VIPSenders, maxResults)
	rpt.Attachments = buildAttachments(emails, maxResults)
	rpt.Threads = buildThreads(emails, maxResults, 3)

	// Fetch today's calendar events
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	events, err := e.Calendar.ListEvents(ctx, domain.ListEventsParams{
		Start:      dayStart,
		End:        dayEnd,
		MaxResults: maxResults,
	})
	if err != nil {
		// Non-fatal: log and return empty calendar section
		e.Logger.Warn("failed to fetch calendar events", slog.Any("error", err))
		events = nil
	}
	rpt.Events = buildCalendar(events, emails, maxResults)

	return rpt, nil
}
