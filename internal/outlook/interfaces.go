package outlook

import "github.com/isai/outlook-mcp/internal/domain"

type MailStore = domain.MailStore

type CalendarStore = domain.CalendarStore

type OutlookSession interface {
	Connect() error
	Close() error
	IsConnected() bool
}
