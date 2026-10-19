// Package main provides the optional legacy command for saved Specrelay prompts.
package main

import (
	"github.com/DeeliN221/specrelay/internal/specrelaycli"
	"os"
)

func main() { os.Exit(specrelaycli.Main(os.Args[1:], os.Stdout, os.Stderr)) }
