package settings

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestProviderEnvNamesCoverTheProviders(t *testing.T) {
	for _, tc := range []struct {
		namespace string
		want      []string
	}{
		{"llm-deepseek", []string{"DEEPSEEK_API_KEY"}},
		{"llm-deepseek-api-key", []string{"DEEPSEEK_API_KEY"}},
		{"llm-openai", []string{"OPENAI_API_KEY"}},
		{"llm-anthropic", []string{"ANTHROPIC_API_KEY"}},
		{"llm-google", []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"}},
		{"llm-pi-ai", nil},
		{"llm-deepseek-account", nil},
		{"llm-vision-fallback", nil},
		{"", nil},
		{"not-a-provider", nil},
	} {
		if got := ProviderEnvNames(tc.namespace); !slices.Equal(got, tc.want) {
			t.Errorf("ProviderEnvNames(%q) = %v, want %v", tc.namespace, got, tc.want)
		}
	}
}

func TestProviderEnvNamesAreACopy(t *testing.T) {
	names := ProviderEnvNames("llm-deepseek")
	if len(names) == 0 {
		t.Fatal("no names")
	}
	names[0] = "MUTATED"
	if got := ProviderEnvNames("llm-deepseek"); got[0] != "DEEPSEEK_API_KEY" {
		t.Fatalf("the table was mutated: %v", got)
	}
}

func TestProviderEnvReadsTheProcess(t *testing.T) {
	s := New()
	t.Setenv("DEEPSEEK_API_KEY", "")
	names, present := s.ProviderEnv("llm-deepseek")
	if present {
		t.Fatal("a blank variable reports a key")
	}
	if !slices.Equal(names, []string{"DEEPSEEK_API_KEY"}) {
		t.Fatalf("names = %v", names)
	}
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	if _, present := s.ProviderEnv("llm-deepseek"); !present {
		t.Fatal("the exported key was not found")
	}
}

func TestProviderEnvAcceptsAnyOfSeveralNames(t *testing.T) {
	s := New()
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	if _, present := s.ProviderEnv("llm-google"); present {
		t.Fatal("no variable is set, but a key was reported")
	}
	t.Setenv("GOOGLE_API_KEY", "google-key")
	if _, present := s.ProviderEnv("llm-google"); !present {
		t.Fatal("the fallback variable was not found")
	}
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	if _, present := s.ProviderEnv("llm-google"); !present {
		t.Fatal("the preferred variable was not found")
	}
}

func TestProviderEnvIgnoresBlankValues(t *testing.T) {
	s := New()
	t.Setenv("DEEPSEEK_API_KEY", "   ")
	if _, present := s.ProviderEnv("llm-deepseek"); present {
		t.Fatal("whitespace counts as a key")
	}
}

func TestProviderEnvUnknownNamespace(t *testing.T) {
	s := New()
	if names, present := s.ProviderEnv("not-a-provider"); len(names) != 0 || present {
		t.Fatalf("unknown namespace reported names=%v present=%v", names, present)
	}
}

func TestProviderEnvPrefersTheDeclaredVariable(t *testing.T) {
	s := New()
	s.namespaces = []Namespace{{
		NS:    "llm-acme",
		Value: json.RawMessage(`{"apiKeyEnv":"ACME_API_KEY"}`),
	}}
	t.Setenv("ACME_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	names, present := s.ProviderEnv("llm-acme")
	if !slices.Equal(names, []string{"ACME_API_KEY"}) {
		t.Fatalf("names = %v, want the declared variable only", names)
	}
	if present {
		t.Fatal("the exported default counted for a namespace that renamed its key")
	}
	t.Setenv("ACME_API_KEY", "acme-key")
	if _, present := s.ProviderEnv("llm-acme"); !present {
		t.Fatal("the declared variable was not found")
	}
}

func TestProviderEnvOnANilStore(t *testing.T) {
	var s *Store
	if names, present := s.ProviderEnv("llm-deepseek"); len(names) == 0 || present {
		t.Fatalf("nil store reported names=%v present=%v", names, present)
	}
}
