package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/isai/outlook-mcp/internal/config"
	"github.com/isai/outlook-mcp/internal/domain"
	"github.com/isai/outlook-mcp/internal/security"
	libmcp "github.com/mark3labs/mcp-go/mcp"
)

type Handlers struct {
	Mail     domain.MailStore
	Calendar domain.CalendarStore
	Policy   security.PolicyGate
	Config   *config.Config
	Logger   *slog.Logger
}

type toolErrorData struct {
	Tool      string         `json:"tool,omitempty"`
	Category  string         `json:"category"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

type toolErrorPayload struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    toolErrorData `json:"data"`
}

func (h *Handlers) HandleSearchEmails(ctx context.Context, req libmcp.CallToolRequest) (*libmcp.CallToolResult, error) {
	query, err := requireTrimmedString(req, "query")
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	folder := strings.TrimSpace(req.GetString("folder", "Inbox"))
	if folder == "" {
		folder = "Inbox"
	}
	since, err := optionalRFC3339(req, "since")
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}
	until, err := optionalRFC3339(req, "until")
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}
	maxResults, err := h.optionalMaxResults(req)
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	emails, err := h.Mail.SearchEmails(ctx, domain.SearchEmailsParams{
		Query:      query,
		Folder:     folder,
		Since:      since,
		Until:      until,
		MaxResults: maxResults,
	})
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	items := make([]map[string]any, 0, len(emails))
	for _, email := range emails {
		items = append(items, emailSummary(email))
	}

	return jsonResult(map[string]any{"emails": items, "count": len(items)})
}

func (h *Handlers) HandleGetEmail(ctx context.Context, req libmcp.CallToolRequest) (*libmcp.CallToolResult, error) {
	id, err := requireTrimmedString(req, "id")
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	email, err := h.Mail.GetEmail(ctx, id)
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	return jsonResult(emailDetail(*email))
}

func (h *Handlers) HandleListAttachments(ctx context.Context, req libmcp.CallToolRequest) (*libmcp.CallToolResult, error) {
	emailID, err := requireTrimmedString(req, "email_id")
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	attachments, err := h.Mail.ListAttachments(ctx, domain.ListAttachmentsParams{EmailID: emailID})
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	items := make([]map[string]any, 0, len(attachments))
	for _, attachment := range attachments {
		items = append(items, attachmentDetail(attachment))
	}

	return jsonResult(map[string]any{"email_id": emailID, "attachments": items})
}

func (h *Handlers) HandleCreateDraft(ctx context.Context, req libmcp.CallToolRequest) (*libmcp.CallToolResult, error) {
	params, err := parseCreateDraftParams(req)
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}
	if err := h.checkPolicy("create_draft"); err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	draft, err := h.Mail.CreateDraft(ctx, params)
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	draftID := ""
	if draft != nil {
		draftID = draft.ID
	}
	return jsonResult(map[string]any{"draft_id": draftID, "saved": true})
}

func (h *Handlers) HandleListEvents(ctx context.Context, req libmcp.CallToolRequest) (*libmcp.CallToolResult, error) {
	start, err := requiredRFC3339(req, "start")
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}
	end, err := requiredRFC3339(req, "end")
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}
	maxResults, err := h.optionalMaxResults(req)
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	events, err := h.Calendar.ListEvents(ctx, domain.ListEventsParams{Start: start, End: end, MaxResults: maxResults})
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	items := make([]map[string]any, 0, len(events))
	for _, event := range events {
		items = append(items, eventDetail(event))
	}

	return jsonResult(map[string]any{"events": items, "count": len(items)})
}

func (h *Handlers) HandleGetEvent(ctx context.Context, req libmcp.CallToolRequest) (*libmcp.CallToolResult, error) {
	id, err := requireTrimmedString(req, "id")
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	event, err := h.Calendar.GetEvent(ctx, id)
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	return jsonResult(eventDetail(*event))
}

func (h *Handlers) HandleCreateEvent(ctx context.Context, req libmcp.CallToolRequest) (*libmcp.CallToolResult, error) {
	params, err := parseCreateEventParams(req)
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}
	if err := h.checkPolicy("create_event"); err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	event, err := h.Calendar.CreateEvent(ctx, params)
	if err != nil {
		return h.errorResult(req.Params.Name, err), nil
	}

	eventID := ""
	if event != nil {
		eventID = event.ID
	}
	return jsonResult(map[string]any{"event_id": eventID, "created": true})
}

func parseCreateDraftParams(req libmcp.CallToolRequest) (domain.CreateDraftParams, error) {
	to, err := req.RequireStringSlice("to")
	if err != nil {
		return domain.CreateDraftParams{}, invalidParams("to is required and must be an array of strings")
	}
	recipients := make([]string, 0, len(to))
	for _, recipient := range to {
		trimmed := strings.TrimSpace(recipient)
		if trimmed == "" {
			return domain.CreateDraftParams{}, invalidParams("to must contain at least one non-empty recipient")
		}
		recipients = append(recipients, trimmed)
	}
	if len(recipients) == 0 {
		return domain.CreateDraftParams{}, invalidParams("to must contain at least one recipient")
	}

	subject, err := requireTrimmedString(req, "subject")
	if err != nil {
		return domain.CreateDraftParams{}, err
	}
	body, err := requireTrimmedString(req, "body")
	if err != nil {
		return domain.CreateDraftParams{}, err
	}

	return domain.CreateDraftParams{To: recipients, Subject: subject, Body: body}, nil
}

func parseCreateEventParams(req libmcp.CallToolRequest) (domain.CreateEventParams, error) {
	title, err := requireTrimmedString(req, "title")
	if err != nil {
		return domain.CreateEventParams{}, err
	}
	start, err := requiredRFC3339(req, "start")
	if err != nil {
		return domain.CreateEventParams{}, err
	}
	end, err := requiredRFC3339(req, "end")
	if err != nil {
		return domain.CreateEventParams{}, err
	}
	if !end.After(start) {
		return domain.CreateEventParams{}, invalidParams("end must be after start")
	}

	return domain.CreateEventParams{
		Title:    title,
		Start:    start,
		End:      end,
		Location: strings.TrimSpace(req.GetString("location", "")),
		Body:     strings.TrimSpace(req.GetString("body", "")),
	}, nil
}

func requireTrimmedString(req libmcp.CallToolRequest, key string) (string, error) {
	value, err := req.RequireString(key)
	if err != nil {
		return "", invalidParams(fmt.Sprintf("%s is required", key))
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", invalidParams(fmt.Sprintf("%s must not be empty", key))
	}
	return value, nil
}

func requiredRFC3339(req libmcp.CallToolRequest, key string) (time.Time, error) {
	value, err := requireTrimmedString(req, key)
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, invalidParams(fmt.Sprintf("%s must be a valid RFC3339 timestamp", key))
	}
	return parsed, nil
}

func optionalRFC3339(req libmcp.CallToolRequest, key string) (time.Time, error) {
	value := strings.TrimSpace(req.GetString(key, ""))
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, invalidParams(fmt.Sprintf("%s must be a valid RFC3339 timestamp", key))
	}
	return parsed, nil
}

func (h *Handlers) optionalMaxResults(req libmcp.CallToolRequest) (int, error) {
	args := req.GetArguments()
	if args == nil {
		return h.defaultMaxResults(), nil
	}
	if _, exists := args["max_results"]; !exists {
		return h.defaultMaxResults(), nil
	}
	maxResults, err := req.RequireInt("max_results")
	if err != nil {
		return 0, invalidParams("max_results must be a number between 1 and 500")
	}
	if maxResults < 1 || maxResults > 500 {
		return 0, invalidParams("max_results must be between 1 and 500")
	}
	if configMax := h.defaultMaxResults(); configMax > 0 && maxResults > configMax {
		return configMax, nil
	}
	return maxResults, nil
}

func (h *Handlers) defaultMaxResults() int {
	if h != nil && h.Config != nil && h.Config.Limits.MaxResults > 0 {
		return h.Config.Limits.MaxResults
	}
	return 50
}

func (h *Handlers) checkPolicy(action string) error {
	if h == nil || h.Policy == nil {
		return fmt.Errorf("%w: action denied by policy", domain.ErrPolicyDenied)
	}
	if err := h.Policy.Check(action); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrPolicyDenied, err)
	}
	return nil
}

func jsonResult(payload map[string]any) (*libmcp.CallToolResult, error) {
	return libmcp.NewToolResultJSON(payload)
}

func invalidParams(message string) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalidParams, message)
}

func (h *Handlers) errorResult(tool string, err error) *libmcp.CallToolResult {
	payload := toolErrorPayload{
		Code:    libmcp.INTERNAL_ERROR,
		Message: "internal error",
		Data: toolErrorData{
			Tool:      tool,
			Category:  "internal",
			Retryable: false,
		},
	}

	switch {
	case errors.Is(err, domain.ErrInvalidParams):
		payload.Code = libmcp.INVALID_PARAMS
		payload.Message = err.Error()
		payload.Data.Category = "validation"
	case errors.Is(err, domain.ErrNotFound):
		payload.Code = libmcp.RESOURCE_NOT_FOUND
		payload.Message = "not found"
		payload.Data.Category = "not_found"
	case errors.Is(err, domain.ErrPolicyDenied):
		payload.Message = "action denied by policy"
		payload.Data.Category = "policy_denied"
	case errors.Is(err, domain.ErrNotConnected):
		payload.Message = "Outlook not connected"
		payload.Data.Category = "com_unavailable"
		payload.Data.Retryable = true
	case errors.Is(err, domain.ErrCOMFailure):
		payload.Message = "COM automation failure"
		payload.Data.Category = "com_failure"
		payload.Data.Retryable = true
	default:
		payload.Message = err.Error()
	}

	result, marshalErr := libmcp.NewToolResultJSON(map[string]any{
		"code":    payload.Code,
		"message": payload.Message,
		"data": map[string]any{
			"tool":      payload.Data.Tool,
			"category":  payload.Data.Category,
			"retryable": payload.Data.Retryable,
		},
	})
	if marshalErr != nil {
		return libmcp.NewToolResultError(payload.Message)
	}
	result.IsError = true
	return result
}

func emailSummary(email domain.Email) map[string]any {
	return map[string]any{
		"id":              email.ID,
		"subject":         email.Subject,
		"sender":          email.From,
		"received_at":     email.Date.Format(time.RFC3339),
		"has_attachments": email.HasAttachments,
	}
}

func emailDetail(email domain.Email) map[string]any {
	attachments := make([]map[string]any, 0, len(email.Attachments))
	for _, attachment := range email.Attachments {
		attachments = append(attachments, attachmentDetail(attachment))
	}
	return map[string]any{
		"id":        email.ID,
		"subject":   email.Subject,
		"body_text": email.Body,
		// body_html intentionally omitted: MVP supports plain text only.
		"sender":      email.From,
		"to":          email.To,
		"cc":          email.CC,
		"received_at": email.Date.Format(time.RFC3339),
		"attachments": attachments,
	}
}

func attachmentDetail(attachment domain.Attachment) map[string]any {
	return map[string]any{
		"id":           attachment.ID,
		"name":         attachment.Name,
		"size_bytes":   attachment.Size,
		"content_type": attachment.ContentType,
	}
}

func eventDetail(event domain.CalendarEvent) map[string]any {
	return map[string]any{
		"id":       event.ID,
		"title":    event.Title,
		"start":    event.Start.Format(time.RFC3339),
		"end":      event.End.Format(time.RFC3339),
		"location": event.Location,
		// body_html intentionally omitted: MVP supports plain text only.
		"body_text": event.Body,
	}
}
