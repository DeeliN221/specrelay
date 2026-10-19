// Package app parses deterministic workflow validation command configuration.
package app

import (
	"strings"

	"github.com/DeeliN221/specrelay/internal/i18n"
	"gopkg.in/yaml.v3"
)

// UnmarshalYAML accepts either a bash command string, `run`, or exact argv fields.
func (command *ValidationCommand) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		command.Run = strings.TrimSpace(value.Value)
		if command.Run == "" {
			return i18n.Errorf("validation command string must not be empty")
		}
		command.Executable = ""
		command.Args = nil
		return nil
	case yaml.MappingNode:
		return decodeValidationCommandMap(command, value)
	default:
		return i18n.Errorf("validation command must be a string or mapping")
	}
}

// decodeValidationCommandMap parses one validation command mapping with explicit fields.
func decodeValidationCommandMap(command *ValidationCommand, value *yaml.Node) error {
	var run string
	var executable string
	var args []string
	seen := map[string]bool{}
	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		if seen[key] {
			return i18n.Errorf("validation command field %q is duplicated", key)
		}
		seen[key] = true
		field := value.Content[i+1]
		switch key {
		case "run":
			if err := field.Decode(&run); err != nil {
				return err
			}
		case "executable":
			if err := field.Decode(&executable); err != nil {
				return err
			}
		case "args":
			if err := field.Decode(&args); err != nil {
				return err
			}
		default:
			return i18n.Errorf("validation command unknown field %q", key)
		}
	}
	command.Run = strings.TrimSpace(run)
	command.Executable = strings.TrimSpace(executable)
	command.Args = args
	return nil
}

// validationConfigFromInput validates user-supplied quality gate commands.
func validationConfigFromInput(input validationConfigInput, base ValidationConfig) (ValidationConfig, error) {
	config := ValidationConfig{Commands: cloneValidationCommands(base.Commands), MaxAttemptsPerStage: base.MaxAttemptsPerStage}
	if input.Commands != nil {
		config.Commands = cloneValidationCommands(input.Commands)
	}
	for i, command := range config.Commands {
		if err := validateValidationCommand(i, command); err != nil {
			return ValidationConfig{}, err
		}
	}
	if input.MaxAttemptsPerStage != nil {
		return ValidationConfig{}, i18n.Errorf("validation.max_attempts_per_stage is a removed legacy field; use validation.limit")
	}
	if input.Limit != nil {
		if *input.Limit < 1 {
			return ValidationConfig{}, i18n.Errorf("validation.limit must be positive")
		}
		config.MaxAttemptsPerStage = *input.Limit
	}
	normalizeValidationConfig(&config)
	return config, nil
}

// validateValidationCommand enforces exactly one supported command style.
func validateValidationCommand(index int, command ValidationCommand) error {
	hasRun := strings.TrimSpace(command.Run) != ""
	hasExecutable := strings.TrimSpace(command.Executable) != ""
	hasArgs := len(command.Args) > 0
	if hasRun && (hasExecutable || hasArgs) {
		return i18n.Errorf("validation.commands[%d] cannot set both run and executable/args", index)
	}
	if hasRun {
		return nil
	}
	if !hasExecutable {
		return i18n.Errorf("validation.commands[%d].executable must not be empty", index)
	}
	return nil
}

func cloneValidationCommands(commands []ValidationCommand) []ValidationCommand {
	cloned := make([]ValidationCommand, 0, len(commands))
	for _, command := range commands {
		cloned = append(cloned, ValidationCommand{Run: command.Run, Executable: command.Executable, Args: append([]string(nil), command.Args...)})
	}
	return cloned
}
