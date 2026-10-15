package config

import (
	"encoding/json"
	"math"
)

// PetBehavior holds the native companion's persisted display and interaction
// preferences. False values are intentional and must survive a round trip.
type PetBehavior struct {
	Scale        float64 `json:"scale"`
	Opacity      float64 `json:"opacity"`
	Personality  string  `json:"personality"`
	Activity     string  `json:"activity"`
	SelfTalk     bool    `json:"selfTalk"`
	Wander       bool    `json:"wander"`
	ChatEnabled  bool    `json:"chatEnabled"`
	LockPosition bool    `json:"lockPosition"`
	ShiftToDrag  bool    `json:"shiftToDrag"`
	PowerSave    bool    `json:"powerSave"`
	ClickSound   bool    `json:"clickSound"`
}

// PetPosition is a native window position, including negative coordinates on
// displays to the left of or above the primary display. Nil means unset.
type PetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func DefaultPetBehavior() PetBehavior {
	return PetBehavior{
		Scale: 1, Opacity: 1, Personality: "natural", Activity: "balanced",
		SelfTalk: true, Wander: true, ChatEnabled: true,
	}
}

// NormalizePetBehavior accepts the old DesktopPet literal's zero Behavior as
// defaults. Callers editing preferences should start from DefaultPetBehavior
// or the saved value so that explicit false settings remain distinguishable.
func NormalizePetBehavior(value PetBehavior) PetBehavior {
	if value == (PetBehavior{}) {
		return DefaultPetBehavior()
	}
	return normalizePetBehaviorFields(value)
}

func normalizePetBehaviorFields(value PetBehavior) PetBehavior {
	value.Scale = normalizePetNumber(value.Scale, 0.6, 1.6, 10)
	value.Opacity = normalizePetNumber(value.Opacity, 0.3, 1, 20)
	switch value.Personality {
	case "natural", "genki", "tsundere", "poison":
	default:
		value.Personality = "natural"
	}
	switch value.Activity {
	case "quiet", "balanced", "active":
	default:
		value.Activity = "balanced"
	}
	return value
}

func normalizePetNumber(value, minimum, maximum, steps float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 1
	}
	value = math.Max(minimum, math.Min(maximum, value))
	return math.Round(value*steps) / steps
}

// UnmarshalJSON decodes each preference independently. A malformed field must
// neither reset valid siblings nor turn a default-on interaction off.
func (value *PetBehavior) UnmarshalJSON(data []byte) error {
	*value = DefaultPetBehavior()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	_ = json.Unmarshal(raw["scale"], &value.Scale)
	_ = json.Unmarshal(raw["opacity"], &value.Opacity)
	_ = json.Unmarshal(raw["personality"], &value.Personality)
	_ = json.Unmarshal(raw["activity"], &value.Activity)
	_ = json.Unmarshal(raw["selfTalk"], &value.SelfTalk)
	_ = json.Unmarshal(raw["wander"], &value.Wander)
	_ = json.Unmarshal(raw["chatEnabled"], &value.ChatEnabled)
	_ = json.Unmarshal(raw["lockPosition"], &value.LockPosition)
	_ = json.Unmarshal(raw["shiftToDrag"], &value.ShiftToDrag)
	_ = json.Unmarshal(raw["powerSave"], &value.PowerSave)
	_ = json.Unmarshal(raw["clickSound"], &value.ClickSound)
	*value = normalizePetBehaviorFields(*value)
	return nil
}

func decodePetPosition(data []byte) *PetPosition {
	// Require both coordinates, but allow zero and negative values. A partially
	// decoded or null position must not unexpectedly move a pet to the origin.
	var value struct {
		X *int `json:"x"`
		Y *int `json:"y"`
	}
	if json.Unmarshal(data, &value) != nil || value.X == nil || value.Y == nil {
		return nil
	}
	return &PetPosition{X: *value.X, Y: *value.Y}
}
