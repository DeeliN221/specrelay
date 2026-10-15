// Package main provides the root specrelay command for `go install github.com/DeeliN221/specrelay@latest`.
package main

import (
	"os"

	"github.com/DeeliN221/specrelay/internal/ozcli"
)

// main delegates root-package installs to the specrelay CLI implementation.
func main() {
	os.Exit(ozcli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
