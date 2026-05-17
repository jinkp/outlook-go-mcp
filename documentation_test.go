package main

import (
	"os"
	"strings"
	"testing"
)

func TestREADMEIncludesRequiredSections(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile(README.md) error = %v", err)
	}

	content := string(data)
	requiredSections := []string{
		"# Outlook MCP Server (Go)",
		"## Prerequisites",
		"## Installation",
		"## Configuration",
		"## Available Tools",
		"## Usage with OpenCode",
		"## Usage with Claude Desktop",
		"## Security",
		"## Testing",
		"## Architecture",
	}

	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			t.Fatalf("README.md missing section %q", section)
		}
	}
}

func TestConfigExampleIncludesCommentsAndRequiredFields(t *testing.T) {
	data, err := os.ReadFile("configs/config.example.yaml")
	if err != nil {
		t.Fatalf("ReadFile(configs/config.example.yaml) error = %v", err)
	}

	content := string(data)
	requiredSnippets := []string{
		"# Outlook profile selection.",
		"# Security switches for write-capable operations.",
		"# Storage paths used by the server.",
		"# Structured logging configuration.",
		"# Runtime safety limits.",
		"allow_create_draft: false",
		"allow_create_event: false",
		"attachment_dir: \"C:\\\\OutlookMCP\\\\attachments\"",
		"level: \"info\"",
		"max_results: 50",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("config.example.yaml missing snippet %q", snippet)
		}
	}
}
