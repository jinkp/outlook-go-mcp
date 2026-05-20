// Package opencode handles reading and merging outlook-mcp's MCP entry into opencode.json.
package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Scope selects which opencode.json file to target.
type Scope string

const (
	// ScopeGlobal targets ~/.config/opencode/opencode.json.
	ScopeGlobal Scope = "global"
	// ScopeLocal targets ./opencode.json in the current working directory.
	ScopeLocal Scope = "local"
)

// mcpEntry is the MCP server descriptor written into opencode.json.
// OpenCode requires command to be an array (binary + args combined), not a string.
type mcpEntry struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
}

// GlobalPath returns the path to the global opencode.json file.
func GlobalPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "opencode", "opencode.json"), nil
}

// LocalPath returns the path to the local opencode.json file (./opencode.json).
func LocalPath() string {
	return "opencode.json"
}

// configPath returns the file path for the given scope.
func configPath(scope Scope) (string, error) {
	switch scope {
	case ScopeGlobal:
		return GlobalPath()
	case ScopeLocal:
		return LocalPath(), nil
	default:
		return "", fmt.Errorf("unknown scope %q: must be %q or %q", scope, ScopeGlobal, ScopeLocal)
	}
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

// Save merges {"mcp":{"outlook-mcp":{"type":"local","command":["outlook-mcp","mcp"]}}}
// into the opencode.json file at the path determined by scope, preserving all other keys.
// Creates the file and its parent directory if missing.
func Save(scope Scope) error {
	path, err := configPath(scope)
	if err != nil {
		return err
	}

	// Load existing content (or start fresh)
	root, err := Load(path)
	if err != nil {
		return err
	}

	// Build the outlook-mcp entry.
	// OpenCode requires command as an array: ["binary", "subcommand", ...args].
	entry := mcpEntry{
		Type:    "local",
		Command: []string{"outlook-mcp", "mcp"},
	}

	// Unmarshal the existing mcp map (if present), then set the outlook-mcp key
	mcpMap := make(map[string]json.RawMessage)
	if existing, ok := root["mcp"]; ok {
		if err := json.Unmarshal(existing, &mcpMap); err != nil {
			// mcp key exists but is not an object — overwrite it
			mcpMap = make(map[string]json.RawMessage)
		}
	}

	entryBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal outlook-mcp entry: %w", err)
	}
	mcpMap["outlook-mcp"] = json.RawMessage(entryBytes)

	mcpBytes, err := json.Marshal(mcpMap)
	if err != nil {
		return fmt.Errorf("marshal mcp map: %w", err)
	}
	root["mcp"] = json.RawMessage(mcpBytes)

	// Write back with pretty-print
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal opencode.json: %w", err)
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
