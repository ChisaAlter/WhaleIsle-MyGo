package settings

import (
	"encoding/json"
	"os"
	"strings"
)

// providerCredentialEnv maps a provider namespace to the environment variables
// its API key is read from when the settings do not name one, as the providers
// behind those namespaces document them: the first name is what the model page
// offers, and any of them may satisfy the provider.
//
// A namespace the Host describes may name its own variable in apiKeyEnv, which
// wins over this table. Namespaces without an entry have no such convention:
// llm-pi-ai takes a key per provider profile, llm-deepseek-account signs in
// instead of holding one, and a provider this desktop has never heard of is the
// caller's to name.
var providerCredentialEnv = map[string][]string{
	"llm-deepseek":         {"DEEPSEEK_API_KEY"},
	"llm-deepseek-api-key": {"DEEPSEEK_API_KEY"},
	"llm-openai":           {"OPENAI_API_KEY"},
	"llm-anthropic":        {"ANTHROPIC_API_KEY"},
	"llm-google":           {"GEMINI_API_KEY", "GOOGLE_API_KEY"},
	"llm-gemini":           {"GEMINI_API_KEY", "GOOGLE_API_KEY"},
	"llm-openrouter":       {"OPENROUTER_API_KEY"},
	"llm-groq":             {"GROQ_API_KEY"},
	"llm-mistral":          {"MISTRAL_API_KEY"},
	"llm-xai":              {"XAI_API_KEY"},
	"llm-azure-openai":     {"AZURE_OPENAI_API_KEY"},
}

// ProviderEnvNames returns the environment variables a provider namespace reads
// its API key from, in the order the model page should offer them, or nil when
// the provider has no such convention. The returned slice is a copy the caller
// owns.
func ProviderEnvNames(namespace string) []string {
	names := providerCredentialEnv[namespace]
	if len(names) == 0 {
		return nil
	}
	return append([]string(nil), names...)
}

// ProviderEnv reports the environment variables a provider namespace reads its
// API key from and whether this process has one of them set: the namespace's
// own apiKeyEnv when the Host described it, else the provider's default. A
// variable that is set but blank does not count, and a namespace belonging to no
// provider this desktop knows reports no names and false.
//
// It reads this process's environment as it is now — the same environment the
// harness it launched inherits — not a .env file. A card can therefore say a key
// is missing only when exporting the variable really would supply one.
func (s *Store) ProviderEnv(namespace string) (names []string, present bool) {
	names = ProviderEnvNames(namespace)
	if declared := s.declaredKeyEnv(namespace); declared != "" {
		names = []string{declared}
	}
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
			return names, true
		}
	}
	return names, false
}

// declaredKeyEnv returns the apiKeyEnv the described namespace names, when it
// names one, and "" when it leaves the provider's default in place.
func (s *Store) declaredKeyEnv(namespace string) string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := s.findNamespaceLocked(namespace)
	if ns == nil || len(ns.Value) == 0 {
		return ""
	}
	var value map[string]json.RawMessage
	if json.Unmarshal(ns.Value, &value) != nil {
		return ""
	}
	var name string
	if json.Unmarshal(value["apiKeyEnv"], &name) != nil {
		return ""
	}
	return strings.TrimSpace(name)
}
