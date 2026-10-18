package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"unicode/utf8"
)

// Only the two original individual-Bot objective configuration fields are edited.
type DshbotObjectiveConfig struct {
	Objective string
	MaxRounds int
}
type DshbotObjectiveConfigDraft struct{ Objective, MaxRounds string }
type DshbotObjectiveConfigCommand struct {
	Expected DshbotCapabilityStatus
	Desired  DshbotObjectiveConfig
}

func DshbotObjectiveConfigValue(s DshbotCapabilityStatus) (DshbotObjectiveConfig, error) {
	var out DshbotObjectiveConfig
	text, err := capabilityText(s.Profile, "objective", 2000, false)
	if err != nil {
		return out, err
	}
	var n float64
	raw := s.Profile["objectiveMaxRounds"]
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &n) != nil || n < 1 || n > 200 || n != float64(int(n)) {
		return out, fmt.Errorf("objective round cap is unavailable; do not guess a default")
	}
	return DshbotObjectiveConfig{Objective: text, MaxRounds: int(n)}, nil
}
func (v DshbotObjectiveConfig) Draft() DshbotObjectiveConfigDraft {
	return DshbotObjectiveConfigDraft{Objective: v.Objective, MaxRounds: strconv.Itoa(v.MaxRounds)}
}
func ParseDshbotObjectiveConfigDraft(d DshbotObjectiveConfigDraft) (DshbotObjectiveConfig, error) {
	var out DshbotObjectiveConfig
	if !utf8.ValidString(d.Objective) {
		return out, fmt.Errorf("objective contains invalid Unicode")
	}
	out.Objective = trimBotMemory(d.Objective)
	if out.Objective == "" {
		return out, fmt.Errorf("目标不能为空；清除请使用已有清除目录目标入口 / Use Clear catalog objective to remove it")
	}
	if WhaleTextLength(out.Objective) > 2000 {
		return out, fmt.Errorf("目标最多 2,000 UTF-16 单元 / Objective exceeds 2,000 UTF-16 units")
	}
	n, err := strconv.Atoi(trimBotMemory(d.MaxRounds))
	if err != nil || n < 1 || n > 200 {
		return out, fmt.Errorf("轮数上限须为 1–200 的整数 / Round cap must be an integer from 1 to 200")
	}
	out.MaxRounds = n
	return out, nil
}
func ValidateDshbotObjectiveConfigCommand(cmd DshbotObjectiveConfigCommand) error {
	if err := ValidateDshbotCapabilityProfile(cmd.Expected); err != nil {
		return err
	}
	if cmd.Expected.WriteError != "" {
		return fmt.Errorf("profile cannot be preserved")
	}
	before, err := DshbotObjectiveConfigValue(cmd.Expected)
	if err != nil {
		return err
	}
	parsed, err := ParseDshbotObjectiveConfigDraft(cmd.Desired.Draft())
	if err != nil {
		return err
	}
	if parsed != cmd.Desired {
		return fmt.Errorf("objective must be reviewed after trimming")
	}
	if before == cmd.Desired {
		return fmt.Errorf("objective configuration is unchanged")
	}
	return nil
}
func (c *Client) DshbotSaveObjectiveConfig(ctx context.Context, cmd DshbotObjectiveConfigCommand) (DshbotCapabilityReceipt, error) {
	if err := ValidateDshbotObjectiveConfigCommand(cmd); err != nil {
		return DshbotCapabilityReceipt{}, err
	}
	objective, _ := json.Marshal(cmd.Desired.Objective)
	cap, _ := json.Marshal(cmd.Desired.MaxRounds)
	return c.saveDshbotProfileFields(ctx, cmd.Expected, map[string]json.RawMessage{"objective": objective, "objectiveMaxRounds": cap})
}
