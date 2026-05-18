package mcp

import "testing"

func TestNewServerDoesNotPanic(t *testing.T) {
	handlers := testHandlers()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("NewServer panicked: %v", recovered)
		}
	}()

	server := NewServer(handlers)
	if server == nil {
		t.Fatal("NewServer() = nil")
	}
}

func TestRegisterToolsRegistersExactFifteenTools(t *testing.T) {
	server := NewServer(testHandlers())

	server.RegisterTools()

	registered := server.mcpServer.ListTools()
	if len(registered) != 15 {
		t.Fatalf("len(ListTools()) = %d, want 15", len(registered))
	}
}

func TestServerCanBeConstructed(t *testing.T) {
	server := NewServer(testHandlers())
	if server.handlers == nil {
		t.Fatal("server.handlers = nil")
	}
	if server.mcpServer == nil {
		t.Fatal("server.mcpServer = nil")
	}
}
