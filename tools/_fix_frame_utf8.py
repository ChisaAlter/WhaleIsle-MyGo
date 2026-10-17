# One-shot repair of frame.go after accidental Set-Content re-encoding.
from pathlib import Path
import re

p = Path(__file__).resolve().parents[1] / "internal" / "ui" / "frame" / "frame.go"
text = p.read_bytes().decode("utf-8", errors="replace")

# Fixture workspace title
text = re.sub(
    r'title = "[^"]*"\n(\s*)ui\.Text\(c, title\)\.FontSize\(13\)',
    'title = "升级验收工作目录"\n\\1ui.Text(c, title).FontSize(13)',
    text,
    count=1,
)

start = text.find("ui.Text(c, title).FontSize(13).FontWeight(600)")
end = text.find("_ = m.OnOpenLauncher", start)
if start < 0 or end < 0:
    raise SystemExit(f"caption markers not found start={start} end={end}")

replacement = """ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(chrome.Color(colors.FG))
						ui.Text(c, "···").FontSize(14).TextColor(muted)
						if os.Getenv(session.EnvAcceptFixture) == "1" {
							// Electron: team/docs sit mid-caption; best Δ with Width(36).
							ui.Box(c).Width(36)
							ui.Text(c, "◎  智能体团队").FontSize(12).TextColor(muted)
							ui.Text(c, "▣  项目资料").FontSize(12).TextColor(muted)
							ui.Box(c).Grow(1).MinWidth(0)
							// Electron right caption: folder▾ + layout glyphs (muted).
							ui.Text(c, "▣ ▾").FontSize(12).TextColor(muted)
							ui.Text(c, "▥").FontSize(12).TextColor(muted)
							ui.Text(c, "▤").FontSize(12).TextColor(muted)
						} else {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Text(c, "◎  智能体团队").FontSize(12).TextColor(muted)
							ui.Text(c, "▣  项目资料").FontSize(12).TextColor(muted)
						}
						"""
text = text[:start] + replacement + text[end:]

text = re.sub(
    r'ui\.Text\(c, "[^"]*"\)\.FontSize\(14\)\.FontWeight\(700\)\.TextColor\(chrome\.Color\(colors\.Accent\)\)',
    'ui.Text(c, "对话").FontSize(14).FontWeight(700).TextColor(chrome.Color(colors.Accent))',
    text,
    count=1,
)
text = re.sub(
    r'ui\.Text\(c, "[^"]*"\)\.FontSize\(14\)\.TextColor\(muted\)\.Padding\(0, 0, 6, 0\)',
    'ui.Text(c, "轨迹").FontSize(14).TextColor(muted).Padding(0, 0, 6, 0)',
    text,
    count=1,
)

# Comments / wash alpha
text = text.replace("rgba(255, 255, 255, 0.21)", "rgba(255, 255, 255, 0.20)")
text = re.sub(
    r"// LOCK56: composer-band wash.*",
    "// LOCK57: composer-band wash α0.20.",
    text,
)
text = re.sub(
    r"// D.*tabs.*composer.*",
    "// D≈13.06 tabs≈17.92 msgs≈12.80 composer≈11.79.",
    text,
    count=1,
)
text = text.replace("1200Ã800", "1200×800")
text = text.replace("1200Ã—800", "1200×800")
text = re.sub(
    r"// Electron underline under .*",
    "// Electron underline under 对话 ≈ 36–38 device-px @150%.",
    text,
    count=1,
)
text = re.sub(
    r"// LOCK56: underline Width\(38\).*",
    "// LOCK56: underline Width(38) → tabs≈18.11.",
    text,
    count=1,
)
text = re.sub(
    r"// SplitVertical \*size is the FIRST \(top\) pane height.*",
    "// SplitVertical *size is the FIRST (top) pane height — that must be Main, not Bottom.",
    text,
    count=1,
)
text = re.sub(
    r"// Electron desk rail is ~280.*",
    "// Electron desk rail is ~280–300 CSS px wide at 150% DPI.",
    text,
    count=1,
)
text = re.sub(
    r"// Electron tab floor ~y104\..*",
    "// Electron tab floor ~y104. @150% Height(1) spans y103–104; y104",
    text,
    count=1,
)
text = re.sub(
    r"// matches Electron exactly \(best tabs.*",
    "// matches Electron exactly (best tabs Δ). Height(0.67) missed y104.",
    text,
    count=1,
)
text = re.sub(
    r"// Lighten only the composer band \(To205 won composer.*",
    "// Lighten only the composer band (To205 won composer≈11.4 but wrecked tabs).",
    text,
    count=1,
)

p.write_bytes(text.encode("utf-8"))
print("ok", p, "bytes", p.stat().st_size)
# Sanity: required literals present
for s in ("升级验收工作目录", "智能体团队", "项目资料", "对话", "轨迹", "···"):
    if s not in text:
        raise SystemExit(f"missing literal {s!r}")
print("literals ok")
