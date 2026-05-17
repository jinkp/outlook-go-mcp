// Package claude handles reading and merging outlook-mcp's MCP entry into Claude Code config files.
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// mcpServer is the MCP server descriptor written into Claude Code config.
type mcpServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// GlobalPath returns the path to the global Claude Code config file (~/.claude.json).
func GlobalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}
	return filepath.Join(home, ".claude.json"), nil
}

// LocalPath returns the path to the local Claude Code config file (./.claude/settings.json).
func LocalPath() (string, error) {
	return filepath.Join(".claude", "settings.json"), nil
}

// Load reads path as a JSON object and returns it as map[string]json.RawMessage.
// If the file is missing or empty, an empty map is returned (not an error).
// If the file contains malformed JSON, an error is returned.
func Load(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]json.RawMessage), nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	if len(data) == 0 {
		return make(map[string]json.RawMessage), nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: invalid JSON: %w", path, err)
	}

	if m == nil {
		return make(map[string]json.RawMessage), nil
	}

	return m, nil
}

// Save merges {"mcpServers":{"outlook-mcp":{"command":"outlook-mcp","args":["mcp"]}}}
// into the Claude Code config file at the given path, preserving all other keys.
// Creates the file and its parent directory if missing.
func Save(path string) error {
	// Load existing content (or start fresh)
	root, err := Load(path)
	if err != nil {
		return err
	}

	// Build the outlook-mcp MCP server entry
	entry := mcpServer{
		Command: "outlook-mcp",
		Args:    []string{"mcp"},
	}

	// Unmarshal the existing mcpServers map (if present), then set the outlook-mcp key
	mcpServersMap := make(map[string]json.RawMessage)
	if existing, ok := root["mcpServers"]; ok {
		if err := json.Unmarshal(existing, &mcpServersMap); err != nil {
			// mcpServers key exists but is not an object — overwrite it
			mcpServersMap = make(map[string]json.RawMessage)
		}
	}

	entryBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal outlook-mcp entry: %w", err)
	}
	mcpServersMap["outlook-mcp"] = json.RawMessage(entryBytes)

	mcpServersBytes, err := json.Marshal(mcpServersMap)
	if err != nil {
		return fmt.Errorf("marshal mcpServers map: %w", err)
	}
	root["mcpServers"] = json.RawMessage(mcpServersBytes)

	// Write back with pretty-print
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal claude config: %w", err)
	}

	// Ensure parent directory exists
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
