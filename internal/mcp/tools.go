package mcp

import libmcp "github.com/mark3labs/mcp-go/mcp"

const iso8601Description = "ISO8601/RFC3339 timestamp, for example 2026-05-16T09:00:00Z."

func ToolDefinitions() []libmcp.Tool {
	return []libmcp.Tool{
		libmcp.NewTool("search_emails",
			libmcp.WithDescription("Search Outlook emails by natural-language query, optionally constrained by folder, time range, and result limit. Use this when the user wants to find messages without needing full message bodies."),
			libmcp.WithString("query", libmcp.Required(), libmcp.MinLength(1), libmcp.Description("Search terms to match against Outlook email content and metadata.")),
			libmcp.WithString("folder", libmcp.DefaultString("Inbox"), libmcp.Description("Outlook mail folder to search. Defaults to Inbox when omitted.")),
			libmcp.WithString("since", libmcp.Description("Only include emails received on or after this timestamp. "+iso8601Description)),
			libmcp.WithString("until", libmcp.Description("Only include emails received on or before this timestamp. "+iso8601Description)),
			libmcp.WithNumber("max_results", libmcp.Min(1), libmcp.Max(500), libmcp.Description("Maximum number of matching emails to return, between 1 and 500.")),
		),
		libmcp.NewTool("get_email",
			libmcp.WithDescription("Fetch a single Outlook email by its unique identifier, including plain-text body and attachment metadata when available."),
			libmcp.WithString("id", libmcp.Required(), libmcp.MinLength(1), libmcp.Description("Unique Outlook identifier of the email to retrieve.")),
		),
		libmcp.NewTool("list_attachments",
			libmcp.WithDescription("List attachment metadata for a specific Outlook email without downloading attachment content."),
			libmcp.WithString("email_id", libmcp.Required(), libmcp.MinLength(1), libmcp.Description("Unique Outlook identifier of the email whose attachments should be listed.")),
		),
		libmcp.NewTool("create_draft",
			libmcp.WithDescription("Create and save a draft email in Outlook. This writes data, so it is subject to policy approval before any Outlook change is made."),
			libmcp.WithArray("to", libmcp.Required(), libmcp.MinItems(1), libmcp.WithStringItems(libmcp.MinLength(1), libmcp.Description("Recipient email address.")), libmcp.Description("List of recipient email addresses for the draft.")),
			libmcp.WithString("subject", libmcp.Required(), libmcp.MinLength(1), libmcp.Description("Subject line for the draft email.")),
			libmcp.WithString("body", libmcp.Required(), libmcp.MinLength(1), libmcp.Description("Plain-text body content for the draft email.")),
		),
		libmcp.NewTool("list_events",
			libmcp.WithDescription("List Outlook calendar events in a required time window. Use this for schedule lookups, availability review, or event discovery."),
			libmcp.WithString("start", libmcp.Required(), libmcp.Description("Only include events starting on or after this timestamp. "+iso8601Description)),
			libmcp.WithString("end", libmcp.Required(), libmcp.Description("Only include events ending on or before this timestamp. "+iso8601Description)),
			libmcp.WithNumber("max_results", libmcp.Min(1), libmcp.Max(500), libmcp.Description("Maximum number of events to return, between 1 and 500.")),
		),
		libmcp.NewTool("get_event",
			libmcp.WithDescription("Fetch one Outlook calendar event by its unique identifier, including timing, location, and plain-text body when available."),
			libmcp.WithString("id", libmcp.Required(), libmcp.MinLength(1), libmcp.Description("Unique Outlook identifier of the calendar event to retrieve.")),
		),
		libmcp.NewTool("create_event",
			libmcp.WithDescription("Create a simple Outlook calendar event with start and end times. This is for non-meeting events only and is subject to policy approval before writing to Outlook."),
			libmcp.WithString("title", libmcp.Required(), libmcp.MinLength(1), libmcp.Description("Title shown on the calendar event.")),
			libmcp.WithString("start", libmcp.Required(), libmcp.Description("Event start timestamp. "+iso8601Description)),
			libmcp.WithString("end", libmcp.Required(), libmcp.Description("Event end timestamp. Must be after start. "+iso8601Description)),
			libmcp.WithString("location", libmcp.Description("Optional physical or virtual location for the event.")),
			libmcp.WithString("body", libmcp.Description("Optional plain-text notes or description for the event.")),
		),
	}
}
