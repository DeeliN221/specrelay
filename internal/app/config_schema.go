// Package app defines the YAML schema inputs used to build workflow configs.
package app

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var iterationStage = regexp.MustCompile(`^(repair|review|qa|fix)_([1-9][0-9]*)$`)

type woConfigFile struct {
	MC woConfig `yaml:"wo"`
}

type woConfig struct {
	Workflow workflowConfigInput `yaml:"workflow"`
	Prompts  map[string]string   `yaml:"prompts"`
}

type workflowConfigInput struct {
	Engine              string                       `yaml:"engine"`
	MaxAuditIterations  *int                         `yaml:"max_audit_iterations"`
	MaxRepairIterations *int                         `yaml:"max_repair_iterations"`
	MaxReviewIterations *int                         `yaml:"max_review_iterations"`
	SubagentGuard       subagentGuardModeInput       `yaml:"subagent_guard"`
	Defaults            stageOptionsInput            `yaml:"defaults"`
	Stages              map[string]stageOptionsInput `yaml:"stages"`
	Iterations          map[string]stageOptionsInput `yaml:"iterations"`
	Parallel            parallelConfigInput          `yaml:"parallel"`
	Subagents           parallelConfigInput          `yaml:"subagents"`
	Validation          validationConfigInput        `yaml:"validation"`
	Prompts             map[string]string            `yaml:"prompts"`
	legacyParallelSet   bool                         `yaml:"-"`
	legacySubagentsSet  bool                         `yaml:"-"`
	legacyGuardSet      bool                         `yaml:"-"`
}

// UnmarshalYAML records deleted workflow keys before typed decoding loses empty/null presence.
func (input *workflowConfigInput) UnmarshalYAML(value *yaml.Node) error {
	type raw workflowConfigInput
	var decoded raw
	if value.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(value.Content); i += 2 {
			switch value.Content[i].Value {
			case "parallel":
				decoded.legacyParallelSet = true
			case "subagents":
				decoded.legacySubagentsSet = true
			case "subagent_guard":
				decoded.legacyGuardSet = true
			}
		}
	}
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*input = workflowConfigInput(decoded)
	return nil
}

type parallelConfigInput struct {
	Enabled *bool                               `yaml:"enabled"`
	Groups  map[string]parallelGroupConfigInput `yaml:"groups"`
}

const (
	subagentGuardModeAdvisory = "advisory"
	subagentGuardModeStrict   = "strict"
	subagentGuardModeOff      = "off"
)

type subagentGuardModeInput struct {
	Set  bool
	Mode string
}

// UnmarshalYAML accepts `subagent_guard: advisory|strict|off` plus bool shorthand.
func (input *subagentGuardModeInput) UnmarshalYAML(value *yaml.Node) error {
	input.Set = true
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("subagent_guard must be advisory, strict, off, or true/false")
	}
	mode := strings.ToLower(strings.TrimSpace(value.Value))
	switch mode {
	case "true", "on", "enabled", subagentGuardModeAdvisory:
		input.Mode = subagentGuardModeAdvisory
	case subagentGuardModeStrict:
		input.Mode = subagentGuardModeStrict
	case "false", "disabled", subagentGuardModeOff:
		input.Mode = subagentGuardModeOff
	default:
		return fmt.Errorf("subagent_guard is invalid: %q", value.Value)
	}
	return nil
}

func normalizeSubagentGuardMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case subagentGuardModeStrict:
		return subagentGuardModeStrict
	case subagentGuardModeOff:
		return subagentGuardModeOff
	default:
		return subagentGuardModeAdvisory
	}
}

func subagentGuardEnabled(mode string) bool {
	return normalizeSubagentGuardMode(mode) != subagentGuardModeOff
}

func subagentGuardStrict(mode string) bool {
	return normalizeSubagentGuardMode(mode) == subagentGuardModeStrict
}

// UnmarshalYAML accepts the KISS `parallel: true|false` scalar config shape.
func (input *parallelConfigInput) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		var enabled bool
		if err := value.Decode(&enabled); err != nil {
			return err
		}
		input.Enabled = &enabled
		return nil
	}
	type raw parallelConfigInput
	var decoded raw
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*input = parallelConfigInput(decoded)
	return nil
}

type parallelGroupConfigInput struct {
	Mode    string                      `yaml:"mode"`
	Members []parallelMemberConfigInput `yaml:"members"`
}

type parallelMemberConfigInput struct {
	Name     string `yaml:"name"`
	Purpose  string `yaml:"purpose"`
	Stage    string `yaml:"stage"`
	CLI      string `yaml:"cli"`
	Tool     string `yaml:"tool"`
	Agent    string `yaml:"agent"`
	Model    string `yaml:"model"`
	Subagent string `yaml:"subagent"`
	Required bool   `yaml:"required"`
}

type validationConfigInput struct {
	Commands            []ValidationCommand `yaml:"commands"`
	MaxAttemptsPerStage *int                `yaml:"max_attempts_per_stage"`
	Limit               *int                `yaml:"limit"`
}

type stageOptionsInput struct {
	CLI         *string                     `yaml:"cli"`
	Tool        *string                     `yaml:"tool"`
	Agent       *string                     `yaml:"agent"`
	Model       *string                     `yaml:"model"`
	Reasoning   *string                     `yaml:"reasoning"`
	Fast        *bool                       `yaml:"fast"`
	Permissions *string                     `yaml:"permissions"`
	Before      []parallelMemberConfigInput `yaml:"before"`
}

// hasValues reports whether a stage override contains any user field.
func (input stageOptionsInput) hasValues() bool {
	return input.CLI != nil || input.Tool != nil || input.Agent != nil || input.Model != nil || input.Reasoning != nil || input.Fast != nil || input.Permissions != nil || input.Before != nil
}

// hasValues reports whether the workflow input contains any user field.
func (input workflowConfigInput) hasValues() bool {
	return input.Engine != "" || input.MaxAuditIterations != nil || input.MaxRepairIterations != nil || input.MaxReviewIterations != nil || input.SubagentGuard.Set || input.Defaults.hasValues() || input.Stages != nil || input.Iterations != nil || input.Parallel.Enabled != nil || input.Parallel.Groups != nil || input.Subagents.Enabled != nil || input.Subagents.Groups != nil || input.Validation.MaxAttemptsPerStage != nil || input.Validation.Limit != nil || input.Validation.Commands != nil
}

func hasLegacyWorkflowRoot(data []byte) (bool, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return false, err
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return false, nil
	}
	for i := 0; i+1 < len(root.Content[0].Content); i += 2 {
		key := root.Content[0].Content[i].Value
		value := root.Content[0].Content[i+1]
		if key == "workflow" {
			return true, nil
		}
		if key == "wo" && mappingHasKey(value, "workflow") {
			return true, nil
		}
	}
	return false, nil
}

func mappingHasKey(node *yaml.Node, key string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return true
		}
	}
	return false
}

// workflowConfigFromInput migrates user YAML into a dynamic quality-loop snapshot.
func workflowConfigFromInput(input workflowConfigInput, baseConfig *WorkflowConfig) (WorkflowConfig, error) {
	if input.Defaults.hasValues() {
		return WorkflowConfig{}, fmt.Errorf("defaults is a removed legacy field")
	}
	if input.Iterations != nil {
		return WorkflowConfig{}, fmt.Errorf("iterations is a removed legacy field")
	}
	if input.legacySubagentsSet {
		return WorkflowConfig{}, fmt.Errorf("subagents is a removed legacy field")
	}
	if input.legacyParallelSet {
		return WorkflowConfig{}, fmt.Errorf("parallel is a removed legacy field")
	}
	maxIterations := 0
	maxAuditIterations := defaultMaxAuditIterations
	engine := "go-dag"
	var basePrompts map[string]string
	byKind := defaultStageOptionsByKind()
	validation := ValidationConfig{MaxAttemptsPerStage: 3}
	if baseConfig != nil {
		engine = baseConfig.Engine
		maxAuditIterations = baseConfig.MaxAuditIterations
		maxIterations = baseConfig.MaxRepairIterations
		if maxIterations == 0 && baseConfig.MaxReviewIterations > 0 {
			maxIterations = baseConfig.MaxReviewIterations
		}
		basePrompts = clonePrompts(baseConfig.Prompts)
		validation = baseConfig.Validation
		if option, ok := baseConfig.Stages["planning"]; ok {
			byKind["planning"] = option
		}
		if option, ok := baseConfig.Stages["execution"]; ok {
			byKind["execution"] = option
		}
		if option, ok := baseConfig.Stages["fix_1"]; ok {
			byKind["fix"] = option
		}
		if option, ok := baseConfig.Stages["repair_1"]; ok {
			byKind["repair"] = option
		} else if option, ok := baseConfig.Stages["audit_1"]; ok {
			byKind["repair"] = option
		} else if option, ok := baseConfig.Stages["targeted_repair_1"]; ok {
			byKind["repair"] = option
		}
		if option, ok := baseConfig.Stages["review_1"]; ok {
			byKind["review"] = option
		}
		if option, ok := baseConfig.Stages["qa_1"]; ok {
			byKind["qa"] = option
		}
		if option, ok := baseConfig.Stages["archive"]; ok {
			byKind["archive"] = option
		}
	}
	if input.MaxRepairIterations != nil && input.MaxReviewIterations != nil {
		return WorkflowConfig{}, fmt.Errorf("max_repair_iterations cannot coexist with deprecated max_review_iterations")
	}
	if input.MaxAuditIterations != nil {
		if *input.MaxAuditIterations < 0 {
			return WorkflowConfig{}, fmt.Errorf("max_audit_iterations must not be negative")
		}
		maxAuditIterations = *input.MaxAuditIterations
	}
	if input.MaxRepairIterations != nil {
		if *input.MaxRepairIterations < 0 {
			return WorkflowConfig{}, fmt.Errorf("max_repair_iterations must not be negative")
		}
		maxIterations = *input.MaxRepairIterations
	} else if input.MaxReviewIterations != nil {
		if *input.MaxReviewIterations < 0 {
			return WorkflowConfig{}, fmt.Errorf("max_review_iterations must not be negative")
		}
		maxIterations = *input.MaxReviewIterations
	}
	if input.Stages != nil {
		_, hasRepair := input.Stages["repair"]
		_, hasReview := input.Stages["review"]
		_, hasFix := input.Stages["fix"]
		if hasRepair && (hasReview || hasFix) {
			return WorkflowConfig{}, fmt.Errorf("stages.repair cannot coexist with deprecated stages.review/fix")
		}
		if !hasRepair {
			if legacy, ok := input.Stages["fix"]; ok {
				input.Stages["repair"] = legacy
			} else if legacy, ok := input.Stages["review"]; ok {
				input.Stages["repair"] = legacy
			}
		}
	}
	if strings.TrimSpace(input.Engine) != "" {
		return WorkflowConfig{}, fmt.Errorf("engine is a removed legacy field")
	}
	if input.legacyGuardSet {
		return WorkflowConfig{}, fmt.Errorf("subagent_guard is a removed legacy field")
	}
	for _, kind := range stageKinds {
		base := byKind[kind]
		if err := mergeStageOptions(&base, input.Defaults); err != nil {
			return WorkflowConfig{}, fmt.Errorf("defaults: %w", err)
		}
		if input.Stages != nil {
			for key := range input.Stages {
				if !slices.Contains(stageKinds, key) {
					return WorkflowConfig{}, fmt.Errorf("unknown stage type %q", key)
				}
			}
			if override, ok := input.Stages[kind]; ok {
				if override.Before != nil {
					return WorkflowConfig{}, fmt.Errorf("stages.%s.before is a removed legacy field", kind)
				}
				if err := mergeStageOptions(&base, override); err != nil {
					return WorkflowConfig{}, fmt.Errorf("stages.%s: %w", kind, err)
				}
			}
		}
		byKind[kind] = base
	}
	config := WorkflowConfig{Engine: engine, Generation: qualityLoopWorkflowGeneration, MaxAuditIterations: maxAuditIterations, MaxRepairIterations: maxIterations, Stages: map[string]StageOptions{
		"planning":          byKind["planning"],
		"execution":         byKind["execution"],
		"audit_1":           byKind["repair"],
		"qa_1":              byKind["qa"],
		"targeted_repair_1": byKind["repair"],
		"archive":           byKind["archive"],
	}, Prompts: basePrompts}
	if input.MaxRepairIterations != nil {
		config.Warnings = append(config.Warnings, "max_repair_iterations is deprecated; quality-loop-v1 retains this for diagnostics, not as a termination limit")
	}
	if input.MaxReviewIterations != nil || input.Stages["review"].hasValues() || input.Stages["fix"].hasValues() {
		config.Warnings = append(config.Warnings, "max_review_iterations and review/fix configuration are deprecated and migrated to repair")
	}
	validation, err := validationConfigFromInput(input.Validation, validation)
	if err != nil {
		return WorkflowConfig{}, err
	}
	config.Validation = validation
	for key, override := range input.Iterations {
		if !iterationStage.MatchString(key) {
			return WorkflowConfig{}, fmt.Errorf("unknown iteration stage %q", key)
		}
		base, ok := config.Stages[key]
		if !ok {
			return WorkflowConfig{}, fmt.Errorf("iteration stage %q exceeds max_repair_iterations", key)
		}
		if err := mergeStageOptions(&base, override); err != nil {
			return WorkflowConfig{}, fmt.Errorf("iterations.%s: %w", key, err)
		}
		config.Stages[key] = base
	}
	return config, nil
}

func mergeStageOptions(base *StageOptions, override stageOptionsInput) error {
	if override.CLI != nil {
		return fmt.Errorf("cli is a removed legacy field; use agent")
	}
	if override.Tool != nil {
		return fmt.Errorf("tool is a removed legacy field; use agent")
	}
	if override.Permissions != nil {
		return fmt.Errorf("permissions is a removed legacy field")
	}
	if override.Agent != nil {
		override.Tool = override.Agent
	}
	if override.Tool != nil {
		if !validAgentTool(*override.Tool) {
			return fmt.Errorf("unknown agent tool %q", *override.Tool)
		}
		base.Tool = *override.Tool
	}
	if override.Model != nil {
		base.Model = *override.Model
	}
	if override.Reasoning != nil {
		if !validReasoning[*override.Reasoning] {
			return fmt.Errorf("invalid reasoning %q", *override.Reasoning)
		}
		base.Reasoning = *override.Reasoning
	}
	if override.Fast != nil {
		base.Fast = *override.Fast
	}
	if override.Permissions != nil {
		if !validPermissions[*override.Permissions] {
			return fmt.Errorf("invalid permissions %q", *override.Permissions)
		}
		base.Permissions = *override.Permissions
	}
	return nil
}
