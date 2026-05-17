package mcp

import (
	"testing"

	libmcp "github.com/mark3labs/mcp-go/mcp"
)

func TestToolDefinitionsRegistersExactSevenTools(t *testing.T) {
	tools := ToolDefinitions()

	if len(tools) != 7 {
		t.Fatalf("len(ToolDefinitions()) = %d, want 7", len(tools))
	}

	wantNames := map[string]struct{}{
		"search_emails":    {},
		"get_email":        {},
		"list_attachments": {},
		"create_draft":     {},
		"list_events":      {},
		"get_event":        {},
		"create_event":     {},
	}

	for _, tool := range tools {
		if _, ok := wantNames[tool.Name]; !ok {
			t.Fatalf("unexpected tool registered: %q", tool.Name)
		}
		delete(wantNames, tool.Name)
	}

	if len(wantNames) != 0 {
		t.Fatalf("missing tool registrations: %#v", wantNames)
	}

	if findToolByName(tools, "download_attachment") != nil {
		t.Fatal("download_attachment must not be registered")
	}
	if findToolByName(tools, "send_email") != nil {
		t.Fatal("send_email must not be registered")
	}
}

func TestToolDefinitionsMarkRequiredParams(t *testing.T) {
	tools := ToolDefinitions()

	assertRequiredParams(t, findToolByName(tools, "search_emails"), "query")
	assertRequiredParams(t, findToolByName(tools, "get_email"), "id")
	assertRequiredParams(t, findToolByName(tools, "list_attachments"), "email_id")
	assertRequiredParams(t, findToolByName(tools, "create_draft"), "to", "subject", "body")
	assertRequiredParams(t, findToolByName(tools, "list_events"), "start", "end")
	assertRequiredParams(t, findToolByName(tools, "get_event"), "id")
	assertRequiredParams(t, findToolByName(tools, "create_event"), "title", "start", "end")
}

func TestToolDefinitionsApplySchemaBoundsAndDefaults(t *testing.T) {
	tools := ToolDefinitions()

	searchEmails := findToolByName(tools, "search_emails")
	if searchEmails == nil {
		t.Fatal("search_emails tool not found")
	}
	folderSchema := propertySchema(t, *searchEmails, "folder")
	if got := folderSchema["default"]; got != "Inbox" {
		t.Fatalf("search_emails.folder default = %#v, want %q", got, "Inbox")
	}
	maxResultsSchema := propertySchema(t, *searchEmails, "max_results")
	if got := maxResultsSchema["minimum"]; got != 1 {
		t.Fatalf("search_emails.max_results minimum = %#v, want 1", got)
	}
	if got := maxResultsSchema["maximum"]; got != 500 {
		t.Fatalf("search_emails.max_results maximum = %#v, want 500", got)
	}

	listEvents := findToolByName(tools, "list_events")
	if listEvents == nil {
		t.Fatal("list_events tool not found")
	}
	listEventsMaxResults := propertySchema(t, *listEvents, "max_results")
	if got := listEventsMaxResults["maximum"]; got != 500 {
		t.Fatalf("list_events.max_results maximum = %#v, want 500", got)
	}

	createDraft := findToolByName(tools, "create_draft")
	if createDraft == nil {
		t.Fatal("create_draft tool not found")
	}
	toSchema := propertySchema(t, *createDraft, "to")
	if got := toSchema["type"]; got != "array" {
		t.Fatalf("create_draft.to type = %#v, want %q", got, "array")
	}
	if got := toSchema["minItems"]; got != 1 {
		t.Fatalf("create_draft.to minItems = %#v, want 1", got)
	}

	createEvent := findToolByName(tools, "create_event")
	if createEvent == nil {
		t.Fatal("create_event tool not found")
	}
	bodySchema := propertySchema(t, *createEvent, "body")
	if got := bodySchema["type"]; got != "string" {
		t.Fatalf("create_event.body type = %#v, want %q", got, "string")
	}
}

func assertRequiredParams(t *testing.T, tool *libmcp.Tool, want ...string) {
	t.Helper()
	if tool == nil {
		t.Fatal("tool not found")
	}
	required := make(map[string]struct{}, len(tool.InputSchema.Required))
	for _, name := range tool.InputSchema.Required {
		required[name] = struct{}{}
	}
	if len(required) != len(want) {
		t.Fatalf("tool %q required params = %#v, want %#v", tool.Name, tool.InputSchema.Required, want)
	}
	for _, name := range want {
		if _, ok := required[name]; !ok {
			t.Fatalf("tool %q missing required param %q", tool.Name, name)
		}
	}
}

func findToolByName(tools []libmcp.Tool, name string) *libmcp.Tool {
	for i := range tools {
		if tools[i].Name == name {
			return &tools[i]
		}
	}
	return nil
}

func propertySchema(t *testing.T, tool libmcp.Tool, name string) map[string]any {
	t.Helper()
	prop, ok := tool.InputSchema.Properties[name]
	if !ok {
		t.Fatalf("tool %q missing property %q", tool.Name, name)
	}
	schema, ok := prop.(map[string]any)
	if !ok {
		t.Fatalf("tool %q property %q type = %T, want map[string]any", tool.Name, name, prop)
	}
	return schema
}
