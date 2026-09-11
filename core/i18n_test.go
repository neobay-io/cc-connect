package core

import (
	"strings"
	"testing"
)

func TestI18n_DefaultLanguage(t *testing.T) {
	i := NewI18n(LangEnglish)
	got := i.T(MsgStarting)
	if got == "" {
		t.Error("expected non-empty message")
	}
}

func TestI18n_Chinese(t *testing.T) {
	i := NewI18n(LangChinese)
	got := i.T(MsgStarting)
	if got == "" {
		t.Error("expected non-empty message")
	}
	// Should contain Chinese characters, not English
	if got == "⏳ Processing..." {
		t.Error("expected Chinese translation, got English")
	}
}

func TestI18n_FallbackToEnglish(t *testing.T) {
	i := NewI18n(Language("nonexistent"))
	got := i.T(MsgStarting)
	if got == "" {
		t.Error("should fallback to English")
	}
}

func TestI18n_MissingKey(t *testing.T) {
	i := NewI18n(LangEnglish)
	got := i.T(MsgKey("totally_missing_key"))
	if got != "totally_missing_key" {
		t.Fatalf("unexpected missing-key fallback: %q", got)
	}
}

func TestI18n_Tf(t *testing.T) {
	i := NewI18n(LangEnglish)
	got := i.Tf(MsgNameSet, "myname", "abc123")
	if got == "" {
		t.Error("Tf should return non-empty formatted message")
	}
}

func TestI18n_AllKeysHaveEnglish(t *testing.T) {
	for key, langs := range messages {
		if _, ok := langs[LangEnglish]; !ok {
			t.Errorf("message key %q missing English translation", key)
		}
	}
}

func TestI18n_BuiltinSessionCommandsDescribeLocalSemantics(t *testing.T) {
	for _, lang := range []Language{
		LangEnglish,
		LangChinese,
		LangTraditionalChinese,
		LangJapanese,
		LangSpanish,
	} {
		i := NewI18n(lang)
		if got := i.T(MsgBuiltinCmdList); !strings.Contains(got, "cc-connect") {
			t.Errorf("%s list description = %q, want cc-connect scope", lang, got)
		}
		if got := i.T(MsgBuiltinCmdSearch); !strings.Contains(strings.ToLower(got), "message") &&
			!strings.Contains(got, "消息") && !strings.Contains(got, "訊息") &&
			!strings.Contains(got, "メッセージ") && !strings.Contains(strings.ToLower(got), "mensaje") {
			t.Errorf("%s search description = %q, want message-content scope", lang, got)
		}
		if got := i.T(MsgBuiltinCmdDelete); !strings.Contains(got, "Agent transcript") &&
			!strings.Contains(got, "transcript del Agent") {
			t.Errorf("%s delete description = %q, want retained Agent transcript", lang, got)
		}
	}
}
