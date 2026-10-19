package i18n

import (
	"errors"
	"os"
	"testing"
)

func TestParseLanguageAliases(t *testing.T) {
	for _, value := range []string{"", "en", "EN", "en-US", "zh", "zh-CN", "zh_CN", "zh-hans"} {
		if _, err := Parse(value); err != nil {
			t.Errorf("Parse(%q): %v", value, err)
		}
	}
	if _, err := Parse("fr"); err == nil {
		t.Fatal("Parse(fr) unexpectedly succeeded")
	}
}

func TestChineseTranslationFormattingAndFallback(t *testing.T) {
	Register(map[string]string{
		"installed %d skills into %s": "已安装 %d 个技能到 %s",
		"invalid change %q: %w":       "变更 %q 无效：%w",
	})
	SetLanguage(SimplifiedChinese)
	t.Cleanup(func() { SetLanguage(English) })
	if got := Sprintf("installed %d skills into %s", 4, ".agents/skills"); got != "已安装 4 个技能到 .agents/skills" {
		t.Fatalf("Sprintf = %q", got)
	}
	if got := T("unregistered text"); got != "unregistered text" {
		t.Fatalf("fallback = %q", got)
	}
	err := Errorf("invalid change %q: %w", "1-demo", os.ErrNotExist)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Errorf lost wrapping: %v", err)
	}
	if err.Error() != `变更 "1-demo" 无效：file does not exist` {
		t.Fatalf("Errorf = %q", err.Error())
	}
	SetLanguage(English)
	if got := Sprintf("installed %d skills into %s", 4, "x"); got != "installed 4 skills into x" {
		t.Fatalf("english Sprintf = %q", got)
	}
}
