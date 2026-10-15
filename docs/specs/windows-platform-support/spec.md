# Windows platform support

## Purpose

Define portable paths, process invocation, interruption handling, and terminal behavior for sealed workflows on Windows. The [Chinese specification](../../zh-CN/specs/windows-platform-support/spec.md) preserves the earlier wording.

### Requirement: Portable paths

The system MUST use platform-independent path APIs. Persisted relative paths MUST use forward slashes and be converted to native paths when read.

#### Scenario: Persist a Windows stage log

Saving a stage log records a stable slash-separated relative path in state; reopening that state resolves the same file on Windows.

### Requirement: Executable discovery

The system MUST discover Codex, SpecRelay, Git, and configured backends through the platform's executable lookup. npm/pnpm shims such as `codex.cmd` MUST work without requiring a manually specified absolute path.

### Requirement: Direct process invocation

The system MUST pass arguments as an argument vector and multiline prompts through stdin. It MUST NOT concatenate prompts into shell command strings. Newlines and quotes in a prompt MUST reach the child process unchanged.

### Requirement: Interruptible and recoverable execution

On Ctrl-C in Windows Terminal, the system MUST cancel the active child process, flush its JSONL log, and retain an interrupted stage that can be resumed on the next invocation.

### Requirement: Ordinary terminal operation

PowerShell execution MUST work with plain-text status output. SpecRelay MUST NOT require tmux, curses, or Unix-only signals for normal use. Platform-specific process and lock behavior MUST remain behind the platform adapters.
