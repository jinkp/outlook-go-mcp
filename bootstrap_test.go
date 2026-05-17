package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProjectBootstrapArtifacts(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current file path")
	}

	root := filepath.Dir(currentFile)

	modBytes, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	if !strings.Contains(string(modBytes), "module github.com/isai/outlook-mcp") {
		t.Fatalf("go.mod does not declare expected module path")
	}

	requiredDirs := []string{
		filepath.Join(root, "cmd", "outlook-mcp"),
		filepath.Join(root, "internal", "config"),
		filepath.Join(root, "internal", "logging"),
		filepath.Join(root, "internal", "security"),
		filepath.Join(root, "internal", "mcp"),
		filepath.Join(root, "internal", "outlook"),
		filepath.Join(root, "configs"),
	}

	for _, dir := range requiredDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("required directory %q missing: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("required path %q is not a directory", dir)
		}
	}
}
