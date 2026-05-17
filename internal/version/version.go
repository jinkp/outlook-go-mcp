// Package version holds the binary version, injected at build time via ldflags.
package version

// Version is set by the release build:
//
//	go build -ldflags "-X github.com/jinkp/outlook-go-mcp/internal/version.Version=v1.2.3"
//
// Falls back to "dev" for local builds.
var Version = "dev"
