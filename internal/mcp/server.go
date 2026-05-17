package mcp

import (
	"context"
	"os"

	libmcp "github.com/mark3labs/mcp-go/mcp"
	serverpkg "github.com/mark3labs/mcp-go/server"
)

type Server struct {
	handlers  *Handlers
	mcpServer *serverpkg.MCPServer
}

func NewServer(handlers *Handlers) *Server {
	return &Server{
		handlers:  handlers,
		mcpServer: serverpkg.NewMCPServer("outlook-go-mcp", "0.1.0", serverpkg.WithToolCapabilities(false)),
	}
}

func (s *Server) RegisterTools() {
	for _, tool := range ToolDefinitions() {
		s.mcpServer.AddTool(tool, s.handlerFor(tool.Name))
	}
}

func (s *Server) Serve(ctx context.Context) error {
	stdio := serverpkg.NewStdioServer(s.mcpServer)
	return stdio.Listen(ctx, os.Stdin, os.Stdout)
}

func (s *Server) handlerFor(name string) func(context.Context, libmcp.CallToolRequest) (*libmcp.CallToolResult, error) {
	switch name {
	case "search_emails":
		return s.handlers.HandleSearchEmails
	case "get_email":
		return s.handlers.HandleGetEmail
	case "list_attachments":
		return s.handlers.HandleListAttachments
	case "create_draft":
		return s.handlers.HandleCreateDraft
	case "list_events":
		return s.handlers.HandleListEvents
	case "get_event":
		return s.handlers.HandleGetEvent
	case "create_event":
		return s.handlers.HandleCreateEvent
	default:
		panic("unknown tool handler: " + name)
	}
}
