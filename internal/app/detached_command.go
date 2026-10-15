// Package app builds detached specrelay flow worker command lines.
package app

// flowWorkerCommandArgs prefixes internal worker commands with the public specrelay flow group.
func flowWorkerCommandArgs(args ...string) []string {
	return append([]string{"flow"}, args...)
}
