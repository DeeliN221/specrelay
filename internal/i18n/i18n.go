// Package i18n selects the user-visible language and translates registered messages.
package i18n

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Language identifies one supported user-visible language.
type Language string

const (
	// English is the default language for CLI output, skills, and flow prompts.
	English Language = "en"
	// SimplifiedChinese switches CLI output, skills, and flow prompts to zh-CN.
	SimplifiedChinese Language = "zh-CN"
)

var (
	mu       sync.RWMutex
	current  = English
	messages = map[string]string{}
)

// Parse accepts the CLI language names and common aliases.
func Parse(value string) (Language, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "en", "en-us", "en_us":
		return English, nil
	case "zh", "zh-cn", "zh_cn", "zh-hans", "cn":
		return SimplifiedChinese, nil
	default:
		return "", Errorf("unsupported language %q; use en or zh-CN", value)
	}
}

// SetLanguage changes the language used by subsequent translations.
func SetLanguage(language Language) {
	mu.Lock()
	defer mu.Unlock()
	if language == SimplifiedChinese {
		current = SimplifiedChinese
		return
	}
	current = English
}

// Current returns the active language.
func Current() Language {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// IsChinese reports whether translations are currently rendered in Chinese.
func IsChinese() bool {
	return Current() == SimplifiedChinese
}

// Register adds Chinese translations keyed by their English source text.
func Register(catalog map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	for source, translated := range catalog {
		messages[source] = translated
	}
}

// T translates one complete English message, falling back to the source text.
func T(text string) string {
	mu.RLock()
	language := current
	translated, ok := messages[text]
	mu.RUnlock()
	if language == SimplifiedChinese && ok {
		return translated
	}
	return text
}

// Sprintf translates a format string before formatting it.
func Sprintf(format string, args ...any) string {
	return fmt.Sprintf(T(format), args...)
}

// Errorf translates a format string and keeps fmt.Errorf wrapping semantics.
func Errorf(format string, args ...any) error {
	return fmt.Errorf(T(format), args...)
}

// New translates a static error message.
func New(text string) error {
	return errors.New(T(text))
}

// Fprintf translates a format string before writing it.
func Fprintf(w io.Writer, format string, args ...any) (int, error) {
	return fmt.Fprintf(w, T(format), args...)
}

// Fprint writes arguments unchanged; callers pass already-translated dynamic values.
func Fprint(w io.Writer, args ...any) (int, error) {
	return fmt.Fprint(w, args...)
}

// Fprintln translates a single static string argument before writing it.
func Fprintln(w io.Writer, args ...any) (int, error) {
	if len(args) == 1 {
		if text, ok := args[0].(string); ok {
			args[0] = T(text)
		}
	}
	return fmt.Fprintln(w, args...)
}
