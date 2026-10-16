package config

import (
	"encoding/json"
	"fmt"
	"math"
)

// OriginalPetDraft reads the e9f584be Electron schemas without changing either
// saved namespace. skinIDs maps original manifest IDs to validated native
// directory IDs; callers must obtain it from the local safe skin catalog.
func (s *Store) OriginalPetDraft(base DesktopPet, skinIDs map[string]string) (DesktopPet, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report := []string{"来源 / Source: " + s.path + " · live2dPet.settings, live2dPet.x/y, pet.petId"}
	var live, sprite map[string]json.RawMessage
	_ = json.Unmarshal(s.raw["live2dPet"], &live)
	_ = json.Unmarshal(s.raw["pet"], &sprite)
	var settings map[string]json.RawMessage
	if json.Unmarshal(live["settings"], &settings) == nil && len(settings) > 0 {
		var behavior PetBehavior
		if json.Unmarshal(live["settings"], &behavior) == nil {
			base.Behavior = behavior
			report = append(report, fmt.Sprintf("外观/行为草稿已载入（按原版范围归一化）/ Appearance and behavior loaded: scale %.1f, opacity %.0f%%", behavior.Scale, behavior.Opacity*100))
		}
	} else {
		report = append(report, "无可用原版行为设置，保留草稿 / No original behavior settings; draft retained")
	}
	x, xOK := originalPetCoordinate(live["x"])
	y, yOK := originalPetCoordinate(live["y"])
	if xOK && yOK {
		base.Position = &PetPosition{X: x, Y: y}
		report = append(report, fmt.Sprintf("位置草稿 / Position draft: (%d, %d); 保存后按当前显示器工作区限制 / clamped to current displays after Save", x, y))
	} else {
		report = append(report, "无有效 live2dPet.x/y，位置保留；pet.xRatio/yRatio 是窗口内比例，无法转换桌面位置 / No desktop coordinates; window-relative ratios cannot be imported")
	}
	var id string
	_ = json.Unmarshal(sprite["petId"], &id)
	if nativeID, ok := skinIDs[id]; id != "" && ok {
		base.SkinID = nativeID
		report = append(report, fmt.Sprintf("已验证 Codex 皮肤 / Validated Codex skin: %s → %s", id, nativeID))
	} else {
		report = append(report, fmt.Sprintf("皮肤保留：pet.petId=%q 未匹配唯一有效的本地 Codex 图集；Live2D/THA4 不是可转换的图集皮肤 / Skin retained: no unique validated Codex atlas match; Live2D/THA4 is not a sprite skin", id))
	}
	report = append(report, "仅载入草稿，检查后保存；显示开关与 THA4 选择不变，原版配置不改写 / Draft only: review then Save; visibility and THA4 stay unchanged, original configuration preserved")
	return base, report
}

func originalPetCoordinate(raw json.RawMessage) (int, bool) {
	var n *float64
	if json.Unmarshal(raw, &n) != nil || n == nil || math.IsNaN(*n) || math.IsInf(*n, 0) || math.Abs(*n) > 1<<20 {
		return 0, false
	}
	// Electron normalizeLive2dPetState uses Math.round, including negative ties.
	return int(math.Floor(*n + 0.5)), true
}
