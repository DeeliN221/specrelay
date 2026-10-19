// Package app implements specrelay CLI-backed change discovery and status helpers.
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/DeeliN221/specrelay/internal/i18n"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	specrelayCommand       = "specrelay"
	specrelayCommandPrefix []string
)

// Change describes one active specrelay change directory.
type Change struct {
	Name string
	Path string
}

type specrelayListResponse struct {
	Changes []struct {
		Name string `json:"name"`
	} `json:"changes"`
}

type specrelayValidateResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

// ListChanges asks specrelay for active changes and returns them in runner order.
func ListChanges(repo string) ([]Change, error) {
	var out specrelayListResponse
	if err := runSpecrelayJSON(repo, []string{"list", "--json"}, &out); err != nil {
		return nil, err
	}
	changes := make([]Change, 0, len(out.Changes))
	for _, item := range out.Changes {
		if item.Name == "" {
			continue
		}
		if err := validateChangeNameForPath(item.Name); err != nil {
			return nil, err
		}
		changes = append(changes, Change{Name: item.Name, Path: changePath(repo, item.Name)})
	}
	return changes, nil
}

// ValidateChange verifies an specrelay change through the specrelay validate JSON contract.
func ValidateChange(repo, changeName string) error {
	if err := validateChangeNameForPath(changeName); err != nil {
		return err
	}
	var out specrelayValidateResponse
	cmdErr := runSpecrelayJSON(repo, []string{"validate", changeName, "--json"}, &out)
	if cmdErr != nil && len(out.Errors) == 0 {
		return cmdErr
	}
	if !out.Valid {
		if message := strings.Join(out.Errors, "; "); message != "" {
			return i18n.Errorf("%s is not a valid specrelay change: %s", changeName, message)
		}
		return i18n.Errorf("%s is not a valid specrelay change", changeName)
	}
	return cmdErr
}

func runSpecrelayJSON(repo string, args []string, target any) error {
	path, err := resolveCommand(specrelayCommand)
	if err != nil {
		return err
	}
	commandArgs := append([]string{}, specrelayCommandPrefix...)
	commandArgs = append(commandArgs, args...)
	cmd := commandContext(context.Background(), path, commandArgs...)
	cmd.Dir = repo
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmdErr := cmd.Run()
	if stdout.Len() > 0 {
		if err := json.Unmarshal(stdout.Bytes(), target); err != nil {
			return i18n.Errorf("parse specrelay %s JSON failed: %w", strings.Join(args, " "), err)
		}
	}
	if cmdErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = cmdErr.Error()
		}
		return i18n.Errorf("specrelay %s execution failed: %s", strings.Join(args, " "), message)
	}
	if stdout.Len() == 0 {
		return i18n.Errorf("specrelay %s did not output JSON", strings.Join(args, " "))
	}
	return nil
}

func changePath(repo, name string) string {
	return filepath.Join(repo, "docs", "changes", name)
}

func validateChangeNameForPath(name string) error {
	if strings.TrimSpace(name) == "" {
		return i18n.Errorf("change name must not be empty")
	}
	if filepath.IsAbs(name) || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return i18n.Errorf("change name %q contains an invalid path segment", name)
	}
	return nil
}

// ParseChangeSelection converts one-based menu input into unique selected changes.
func ParseChangeSelection(input string, changes []Change) ([]Change, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, i18n.Errorf("invalid selection")
	}
	if strings.EqualFold(trimmed, "a") {
		return append([]Change(nil), changes...), nil
	}
	seen := map[int]bool{}
	var indexes []int
	for _, part := range strings.Split(trimmed, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, i18n.Errorf("invalid selection")
		}
		if strings.Contains(part, "-") {
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return nil, i18n.Errorf("invalid selection")
			}
			start, err := parseSelectionNumber(bounds[0], len(changes))
			if err != nil {
				return nil, err
			}
			end, err := parseSelectionNumber(bounds[1], len(changes))
			if err != nil {
				return nil, err
			}
			if start > end {
				return nil, i18n.Errorf("invalid selection")
			}
			for i := start; i <= end; i++ {
				if !seen[i] {
					seen[i] = true
					indexes = append(indexes, i)
				}
			}
			continue
		}
		n, err := parseSelectionNumber(part, len(changes))
		if err != nil {
			return nil, err
		}
		if !seen[n] {
			seen[n] = true
			indexes = append(indexes, n)
		}
	}
	selected := make([]Change, 0, len(indexes))
	for _, index := range indexes {
		selected = append(selected, changes[index-1])
	}
	return selected, nil
}

// SortChangesByNumericPrefix orders numbered changes before unnumbered changes.
func SortChangesByNumericPrefix(changes []Change) []Change {
	sorted := append([]Change(nil), changes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left, leftOK := changeNumericPrefix(sorted[i].Name)
		right, rightOK := changeNumericPrefix(sorted[j].Name)
		if leftOK != rightOK {
			return leftOK
		}
		if leftOK && rightOK && left != right {
			return left < right
		}
		return false
	})
	return sorted
}

func parseSelectionNumber(input string, max int) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || n < 1 || n > max {
		return 0, i18n.Errorf("invalid selection")
	}
	return n, nil
}

func changeNumericPrefix(name string) (int, bool) {
	prefix, _, ok := strings.Cut(name, "-")
	if !ok || prefix == "" {
		return 0, false
	}
	n, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, false
	}
	return n, true
}
