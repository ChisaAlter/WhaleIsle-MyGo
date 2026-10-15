package config

import (
	"encoding/json"
	"testing"
)

func TestSavedPetPersonalityRequiresExplicitValidSelection(t *testing.T) {
	cases := []struct {
		raw, want string
		valid     bool
	}{
		{`{}`, "", false},
		{`{"behavior":{"personality":"genki"}}`, "", false},
		{`{"personalityConfigured":false,"behavior":{"personality":"poison"}}`, "", false},
		{`{"personalityConfigured":true,"behavior":{}}`, "", false},
		{`{"personalityConfigured":true,"behavior":{"personality":"invalid"}}`, "", false},
		{`{"personalityConfigured":true,"behavior":{"personality":null}}`, "", false},
		{`{"personalityConfigured":true,"behavior":{"personality":"natural"}}`, "natural", true},
		{`{"personalityConfigured":true,"behavior":{"personality":"genki"}}`, "genki", true},
	}
	for _, test := range cases {
		s := &Store{raw: map[string]json.RawMessage{"mygoDesktopPet": json.RawMessage(test.raw)}}
		got, valid := s.SavedPetPersonality()
		if got != test.want || valid != test.valid {
			t.Fatalf("%s -> %q/%v", test.raw, got, valid)
		}
	}
}
