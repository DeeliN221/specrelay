// Package app validates repair artifacts from the command line.
package app

import (
	"fmt"
	"io"
	"strings"
)

// runValidateRepairArtifact validates one durable repair checkpoint.
func runValidateRepairArtifact(args []string, stdout io.Writer) error {
	if !hasFlag(args, "--artifact") {
		return fmt.Errorf("Usage: specrelay flow validate-repair --artifact <artifact-path> [--json]")
	}
	path, err := requireFlagValue(args, "--artifact")
	if err != nil {
		return err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("repair artifact path must not be empty")
	}
	repair, err := ReadRepair(path)
	if hasFlag(args, "--json") {
		result := reviewValidationResult{Path: path, Valid: err == nil}
		if err != nil {
			result.Error = err.Error()
			result.Code = "repairValidationError"
			_ = writeJSON(stdout, result)
			return err
		}
		result.Decision = repair.Decision
		result.Findings = len(repair.Findings)
		return writeJSON(stdout, result)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "repair artifact is valid: %s (decision=%s, findings=%d)\n", path, repair.Decision, len(repair.Findings))
	return nil
}
