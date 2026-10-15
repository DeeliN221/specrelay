// Package app defines the workflow role table shared by prompt, runtime, and status mapping.
package app

import (
	"fmt"
	"strings"
)

// stageRole describes one business role in the workflow state machine.
type stageRole struct {
	Name       string
	PromptKey  string
	PromptName string
	Session    string
	Label      string
	Iterated   bool
	OptionsKey string
	Default    StageOptions
}

var workflowRoles = []stageRole{
	{Name: "planning", PromptKey: "planning", PromptName: "specrelay-flow-discuss", Session: "planner", Label: "plan", OptionsKey: "planning", Default: StageOptions{Tool: "codex", Reasoning: "xhigh", Fast: true}},
	{Name: "execution", PromptKey: "execution", PromptName: "specrelay-flow-start", Session: "executor", Label: "write", OptionsKey: "execution", Default: StageOptions{Tool: "codex", Reasoning: "low", Fast: false}},
	{Name: "repair", PromptKey: "repair", PromptName: "specrelay-flow-repair", Session: "repairer", Label: "fix", Iterated: true, OptionsKey: "repair", Default: StageOptions{Tool: "codex", Reasoning: "medium", Fast: false}},
	{Name: "audit", PromptKey: "repair", PromptName: "specrelay-flow-repair", Session: "repairer", Label: "audit", Iterated: true, OptionsKey: "repair", Default: StageOptions{Tool: "codex", Reasoning: "medium", Fast: false}},
	{Name: "targeted_repair", PromptKey: "repair", PromptName: "specrelay-flow-repair", Session: "repairer", Label: "fix", Iterated: true, OptionsKey: "repair", Default: StageOptions{Tool: "codex", Reasoning: "medium", Fast: false}},
	{Name: "review", PromptKey: "review", PromptName: "specrelay-flow-review", Session: "reviewer", Label: "review", Iterated: true, OptionsKey: "review", Default: StageOptions{Tool: "codex", Reasoning: "high", Fast: false}},
	{Name: "qa", PromptKey: "qa", PromptName: "specrelay-flow-qa", Session: "qa", Label: "test", Iterated: true, OptionsKey: "qa", Default: StageOptions{Tool: "codex", Reasoning: "high", Fast: false}},
	{Name: "fix", PromptKey: "fix", PromptName: "specrelay-flow-fix", Session: "fixer", Label: "fix", Iterated: true, OptionsKey: "fix", Default: StageOptions{Tool: "codex", Reasoning: "low", Fast: false}},
	{Name: "archive", PromptKey: "archive", PromptName: "specrelay-flow-done", Session: "archiver", Label: "save", OptionsKey: "archive", Default: StageOptions{Tool: "codex", Reasoning: "low", Fast: false}},
}

// roleForStage returns the workflow role that owns an expanded stage name.
func roleForStage(stage string) (stageRole, error) {
	for _, role := range workflowRoles {
		if role.Name == stage {
			return role, nil
		}
		if role.Iterated && strings.HasPrefix(stage, role.Name+"_") {
			return role, nil
		}
	}
	return stageRole{}, fmt.Errorf("unknown stage %q", stage)
}

func roleByName(name string) (stageRole, bool) {
	for _, role := range workflowRoles {
		if role.Name == name {
			return role, true
		}
	}
	return stageRole{}, false
}

func roleByPromptName(name string) (stageRole, bool) {
	for _, role := range workflowRoles {
		if role.PromptName == name {
			return role, true
		}
	}
	return stageRole{}, false
}

func rolePromptKeys() []string {
	keys := make([]string, 0, len(workflowRoles))
	for _, role := range workflowRoles {
		keys = append(keys, role.PromptKey)
	}
	return keys
}

func roleStageKinds() []string {
	kinds := make([]string, 0, len(workflowRoles))
	for _, role := range workflowRoles {
		kinds = append(kinds, role.OptionsKey)
	}
	return kinds
}

// statusRoles returns one display row definition per backend-scoped session role.
func statusRoles() []stageRole {
	roles := make([]stageRole, 0, len(workflowRoles))
	seen := map[string]bool{}
	for _, role := range workflowRoles {
		if seen[role.Session] {
			continue
		}
		seen[role.Session] = true
		roles = append(roles, role)
	}
	return roles
}
