// Package main provides the cmd/specrelay entrypoint used by local builds and release automation.
package main

import (
	"os"

	"github.com/DeeliN221/specrelay/internal/ozcli"
)

// main delegates command behavior to the shared specrelay CLI implementation.
func main() {
	os.Exit(ozcli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
