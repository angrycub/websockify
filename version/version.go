package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	// Injected with ldflags at build time
	tag    string
	commit string
	date   string
)

const (
	unknownVersion = "v0.0.0"
	develSuffix    = "-devel"
)

// Version returns the version string for the application.
// If tag is injected via ldflags, it returns that version.
// Otherwise, it attempts to derive version from VCS info or returns unknown.
func Version() string {
	if tag != "" {
		return ensureVPrefix(tag)
	}

	// Try to get version from VCS info (go install, go run)
	if info, ok := debug.ReadBuildInfo(); ok {
		return buildVersionFromVCS(info)
	}

	return unknownVersion + develSuffix
}

// Tag returns the raw tag without "v" prefix.
func Tag() string {
	version := Version()
	return strings.TrimPrefix(version, "v")
}

// Commit returns the commit hash if available.
func Commit() string {
	if commit != "" {
		return commit
	}

	// Try to get from VCS info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}

	return "unknown"
}

// Date returns the build date if available.
func Date() string {
	if date != "" {
		return date
	}

	// Try to get from VCS info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" {
				return setting.Value
			}
		}
	}

	return "unknown"
}

// Full returns a complete version string with commit and date.
func Full() string {
	version := Version()
	commit := Commit()
	date := Date()

	if commit == "unknown" && date == "unknown" {
		return version
	}

	parts := []string{version}
	if commit != "unknown" {
		if len(commit) > 7 {
			commit = commit[:7]
		}
		parts = append(parts, fmt.Sprintf("commit=%s", commit))
	}
	if date != "unknown" {
		parts = append(parts, fmt.Sprintf("date=%s", date))
	}

	return strings.Join(parts, " ")
}

// ensureVPrefix ensures the version string starts with "v"
func ensureVPrefix(version string) string {
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// buildVersionFromVCS constructs a version from VCS build info
func buildVersionFromVCS(info *debug.BuildInfo) string {
	var revision, modified string
	
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}

	version := unknownVersion + develSuffix
	if revision != "" {
		shortRev := revision
		if len(shortRev) > 7 {
			shortRev = shortRev[:7]
		}
		version += "+" + shortRev
		if modified == "true" {
			version += "-dirty"
		}
	}

	return version
}