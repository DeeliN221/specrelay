// Package main provides the optional legacy command for saved Oz prompts.
package main

import (
	"github.com/DeeliN221/specrelay/internal/ozcli"
	"os"
)

func main() { os.Exit(ozcli.Main(os.Args[1:], os.Stdout, os.Stderr)) }
