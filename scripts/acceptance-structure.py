#!/usr/bin/env python3
"""Structural parity report for one acceptance row, in three coordinate spaces.

Reads the UIA dumps produced by scripts/acceptance-capture.ps1 (geom.json) and
matches every native control to a MyGo control by (type, name). For each match it
reports the DIP rectangle delta and whether the contract's 1-DIP tolerance holds;
unmatched native controls are "missing" and unmatched MyGo controls are "extra".

Coordinates are compared in three registered spaces, because the two clients do
not record the same origin:

  absolute        the recorded ``dip`` rect, unchanged. Native UIA reports screen
                  coordinates while MyGo reports client coordinates, so this space
                  carries the window's screen offset as a constant dx (2800 DIP for
                  a window at screen x=-2800). It is kept only for continuity with
                  earlier rounds; a match here is not a layout claim.
  window-relative every rect minus its own window/pane client origin, so both
                  sides share one origin. This is the space that answers "is the
                  control in the same place in the window".
  caption-relative
                  window-relative minus that side's caption bottom, i.e. the rect
                  measured from where the page content starts under the caption.
                  The caption bottom is the bottom edge of the window-control triad
                  (minimize/maximize/close) detected in the dump, or an explicit
                  override. This space absorbs each side's own caption placement,
                  so it answers "is the page laid out the same below the caption";
                  the difference between the two detected bottoms is reported as
                  ``caption_offset_dip`` and is absorbed, not hidden.

Also emits per-region pixel mismatch when baseline.png/actual.png are present, so
a reviewer sees both the structural and the rendered result in one file.

Usage:
  python scripts/acceptance-structure.py <row-dir>
      [--native-origin X,Y] [--mygo-origin X,Y]   window client origin in DIP
      [--native-caption DIP] [--mygo-caption DIP] caption bottom in DIP
      [--out structure-relative.json]

The origin and caption overrides are for rows whose dump lacks the window node or
the control triad; detection is the default and is always reported.
"""

from __future__ import annotations

import json
import pathlib
import re
import statistics
import sys

from PIL import Image, ImageChops

TOLERANCE_DIP = 1.0

# Accessible names the two apps use for the same control. Electron names the
# button 新建会话 while MyGo names it 新会话 and puts a Text child inside; a name
# mismatch is name drift, not a missing control.
ALIASES = {
    "新建会话": "新会话",
    "探索未至之鲸": "探索未至之境",
    "拖动调整输入框高度": "调整输入框高度",
    "拖动调整输入框宽度": "调整输入框宽度",
    "收起侧边栏": "收起左侧栏",
    "切换右侧栏": "切换右侧面板",
    "展开面板": "展开右侧面板",
    "选择工作区": "选择工作区…",
    "选择工作区": "Composer workspace",
    "搜索会话": "搜索会话…",
    "添加工作区": "选择工作区…",
}

# MyGo labels its layout scaffolding as accessible nodes ("Composer card",
# "Composer draft", …). They are not user-visible controls and have no native
# names, so counting them as "extra" polluted the score: the owner is the
# composer, and the native analogue is the Edit/chip inside it. Scaffolding is
# excluded from missing/extra and reported separately.
SCAFFOLD_NAME = re.compile(r"^Composer\s")

# Native names that only exist as scaffolding on the MyGo side, matched by role
# rather than by name.
ROLE_ALIASES = (
    # (native type, native name prefix, mygo name)
    ("Edit", "描述你想要构建的内容", "Composer draft"),
    ("Edit", "Describe what you want", "Composer draft"),
)

# Window controls in the caption band, in both apps' accessible names. Their
# bottom edge is the measured caption bottom.
CAPTION_CONTROLS = {
    "最小化": "minimize",
    "最大化": "maximize",
    "还原": "restore",
    "关闭": "close",
    "Minimize": "minimize",
    "Maximize": "maximize",
    "Restore": "restore",
    "Close": "close",
}

# A caption node some dumps expose directly (`type` or `name` marker).
CAPTION_NODE = re.compile(r"titlebar|title bar|caption|标题栏|标题条", re.I)

# Regions are expressed in DIP and converted with the row's DPI scale (150 %).
REGIONS_DIP = {
    "sidebar": (0, 0, 285, 860),
    "header": (285, 0, 1440, 60),
    "hero": (285, 60, 1440, 400),
    "composer": (285, 400, 1440, 620),
    "footer": (0, 800, 285, 860),
}

WINDOW_TYPES = {"Window", "Pane"}
CONTAINERS = {("Window", "Whale Isle"), ("Pane", "Whale Isle"), ("Document", "Whale Isle")}

# The caption band's window controls all live in this top strip; a document's own
# 关闭 can sit anywhere and must not define the page origin.
CAPTION_BAND_MAX = 64.0


def key(node: dict) -> tuple[str, str]:
    return (node.get("type", ""), node.get("name", ""))


def base_name(name: str) -> str:
    """Strip the ellipsis/formula suffix the two apps append differently."""
    out = (name or "").strip()
    out = re.sub(r"[…\.]{1,3}$", "", out).strip()
    out = re.sub(r"[（(][^（）()]*[）)]$", "", out).strip()
    return out


def rect(node: dict) -> dict:
    return node["dip"]


def shifted(node: dict, ox: float, oy: float) -> dict:
    r = rect(node)
    return {
        "x": round(r["x"] - ox, 2),
        "y": round(r["y"] - oy, 2),
        "w": round(r["w"], 2),
        "h": round(r["h"], 2),
    }


def detect_window_origin(nodes: list[dict], fallback) -> tuple[float, float, str]:
    for node in nodes:
        if node.get("type") in WINDOW_TYPES and node.get("depth", 0) <= 1:
            r = rect(node)
            return float(r["x"]), float(r["y"]), f"window node {node.get('type')}/{node.get('name')!r} depth {node.get('depth')}"
    for node in nodes:
        if node.get("type") in WINDOW_TYPES:
            r = rect(node)
            return float(r["x"]), float(r["y"]), f"pane node {node.get('name')!r}"
    if fallback:
        return float(fallback[0]), float(fallback[1]), "explicit origin"
    return 0.0, 0.0, "none (assumed 0,0)"


def detect_caption_bottom(nodes: list[dict], oy: float, override) -> tuple[float, str]:
    if override is not None:
        return float(override), "explicit caption bottom"
    bottoms = []
    for node in nodes:
        name = (node.get("name") or "").strip()
        if node.get("type") == "Button" and name in CAPTION_CONTROLS:
            r = rect(node)
            top = float(r["y"] - oy)
            # Only the caption band itself: a document's own 关闭 (dialog, tab) can
            # sit anywhere in the window and must not define the page origin.
            if 0 <= top <= CAPTION_BAND_MAX:
                bottoms.append(top + float(r["h"]))
    if bottoms:
        return max(bottoms), f"window-control triad bottom ({len(bottoms)} buttons in the top {CAPTION_BAND_MAX:g} DIP)"
    for node in nodes:
        if CAPTION_NODE.search(node.get("type", "")) or CAPTION_NODE.search(node.get("name") or ""):
            r = rect(node)
            return float(r["y"] - oy + r["h"]), f"caption node {node.get('name')!r}"
    return 0.0, "none (assumed 0)"


def pair(native_nodes: list[dict], mygo_nodes: list[dict]) -> list[tuple[dict, dict]]:
    """Pair same-key nodes; the closest y (then width) wins within a key."""
    left = sorted(native_nodes, key=lambda n: (rect(n)["y"], rect(n)["x"]))
    right = sorted(mygo_nodes, key=lambda n: (rect(n)["y"], rect(n)["x"]))
    used: set[int] = set()
    out: list[tuple[dict, dict]] = []
    for n in left:
        best = None
        for i, m in enumerate(right):
            if i in used:
                continue
            score = abs(rect(m)["y"] - rect(n)["y"]) + abs(rect(m)["w"] - rect(n)["w"])
            if best is None or score < best[0]:
                best = (score, i)
        if best is not None:
            used.add(best[1])
            out.append((n, right[best[1]]))
    return out


def compare(native: dict, mygo: dict, space: str) -> dict:
    dx = round(mygo["x"] - native["x"], 2)
    dy = round(mygo["y"] - native["y"], 2)
    dw = round(mygo["w"] - native["w"], 2)
    dh = round(mygo["h"] - native["h"], 2)
    return {
        "native": native,
        "mygo": mygo,
        "delta": {"x": dx, "y": dy, "w": dw, "h": dh},
        "within_1dip": all(abs(v) <= TOLERANCE_DIP for v in (dx, dy, dw, dh)),
        "space": space,
    }


def dominant_delta(matched: list[dict]) -> dict | None:
    """The most repeated rounded (dx, dy): a registration bias shows up here."""
    if not matched:
        return None
    counts: dict[tuple[int, int], int] = {}
    for record in matched:
        bucket = (round(record["delta"]["x"]), round(record["delta"]["y"]))
        counts[bucket] = counts.get(bucket, 0) + 1
    (dx, dy), hits = max(counts.items(), key=lambda item: (item[1], -abs(item[0][1])))
    return {"dx": dx, "dy": dy, "count": hits, "of": len(matched)}


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print(__doc__)
        return 2
    row = pathlib.Path(argv[1]).resolve()

    def opt(flag: str):
        return argv[argv.index(flag) + 1] if flag in argv else None

    def point(flag: str):
        value = opt(flag)
        if value is None:
            return None
        parts = value.split(",")
        return (float(parts[0]), float(parts[1]))

    out_name = opt("--out") or "structure.json"

    geom = json.loads((row / "geom.json").read_text(encoding="utf-8-sig"))
    scale = geom["client"]["dpi"] / 96.0
    native_nodes = geom["native"]
    mygo_nodes = geom["mygo"]

    nox, noy, norigin_how = detect_window_origin(native_nodes, point("--native-origin"))
    mox, moy, morigin_how = detect_window_origin(mygo_nodes, point("--mygo-origin"))
    ncaption, ncaption_how = detect_caption_bottom(native_nodes, noy, opt("--native-caption"))
    mcaption, mcaption_how = detect_caption_bottom(mygo_nodes, moy, opt("--mygo-caption"))

    def register(node: dict, side: str, space: str) -> dict:
        ox, oy, caption = (nox, noy, ncaption) if side == "native" else (mox, moy, mcaption)
        if space == "absolute":
            return rect(node)
        if space == "window":
            return shifted(node, ox, oy)
        return shifted(node, ox, oy + caption)

    def index(nodes, space, side):
        out: dict[tuple[str, str], list[dict]] = {}
        for node in nodes:
            if not node.get("name"):
                continue
            copy = dict(node)
            copy["registered"] = register(node, side, space)
            out.setdefault(key(node), []).append(copy)
        return out

    spaces = ("absolute", "window", "caption")
    results: dict[str, dict] = {}

    for space in spaces:
        nb = index(native_nodes, space, "native")
        mb = index(mygo_nodes, space, "mygo")
        matched, missing, extra, outside, scaffolding = [], [], [], [], []
        alias_pairs = set()
        normalized = 0
        for k, native_list in nb.items():
            if k in CONTAINERS:
                continue
            mine = mb.get(k)
            if not mine:
                alias = (k[0], ALIASES.get(k[1], ""))
                if alias[1]:
                    mine = mb.get(alias)
                    if mine:
                        alias_pairs.add((k[1], alias[1]))
            if not mine:
                for native_type, prefix, mygo_name in ROLE_ALIASES:
                    if k[0] == native_type and k[1].startswith(prefix):
                        mine = mb.get((native_type, mygo_name))
                        if mine:
                            alias_pairs.add((k[1], mygo_name))
                        break
            if not mine:
                for mk, mv in mb.items():
                    if mk[0] == k[0] and base_name(mk[1]) == base_name(k[1]):
                        mine = mv
                        normalized += 1
                        break
            if not mine:
                for node in native_list:
                    missing.append({"type": k[0], "name": k[1], "native": node["registered"]})
                continue
            for n, m in pair(native_list, mine):
                record = {"type": n["type"], "name": n["name"], "matched_name": m["name"]}
                record.update(compare(n["registered"], m["registered"], space))
                matched.append(record)
                if not record["within_1dip"]:
                    outside.append(record)
        for k, mygo_list in mb.items():
            if k in CONTAINERS:
                continue
            if SCAFFOLD_NAME.match(k[1] or ""):
                for node in mygo_list:
                    scaffolding.append({"type": k[0], "name": k[1], "mygo": node["registered"]})
                continue
            hit = k in nb or (k[0], ALIASES.get(k[1], "")) in nb
            if not hit:
                hit = any(
                    k[0] == native_type and native_name.startswith(prefix)
                    for native_type, prefix, mygo_name in ROLE_ALIASES
                    if mygo_name == k[1]
                    for native_name in [x for x in nb if x[0] == native_type for x in [x[1]]]
                )
            if not hit:
                hit = any(mk[0] == k[0] and base_name(mk[1]) == base_name(k[1]) for mk in nb)
            if not hit:
                for node in mygo_list:
                    extra.append({"type": k[0], "name": k[1], "mygo": node["registered"]})
        results[space] = {
            "matched": len(matched),
            "within_1dip": sum(1 for m in matched if m["within_1dip"]),
            "outside_1dip": len(outside),
            "missing_native_only": len(missing),
            "extra_mygo_only": len(extra),
            "scaffolding_ignored": [{"type": s["type"], "name": s["name"]} for s in scaffolding],
            "alias_matches": [{"native": a, "mygo": b} for a, b in sorted(alias_pairs)],
            "normalized_matches": normalized,
            "dominant_delta": dominant_delta(matched),
            "within_1dip_names": sorted({m["name"] for m in matched if m["within_1dip"]}),
            "outside_tolerance": sorted(
                outside,
                key=lambda m: -max(abs(m["delta"]["x"]), abs(m["delta"]["y"]), abs(m["delta"]["w"]), abs(m["delta"]["h"])),
            )[:30],
            "missing": missing[:40],
            "extra": extra[:40],
        }

    # Registration cross-check: how far the outliers sit in window space, so an
    # unexpected origin is visible instead of being silently absorbed.
    dy_median_outside = None
    if results["window"]["outside_tolerance"]:
        dy_median_outside = round(
            statistics.median([abs(m["delta"]["y"]) for m in results["window"]["outside_tolerance"]]), 2
        )

    regions = {}
    baseline_path = row / "baseline.png"
    actual_path = row / "actual.png"
    if baseline_path.exists() and actual_path.exists():
        b = Image.open(baseline_path).convert("RGB")
        a = Image.open(actual_path).convert("RGB")
        w = min(b.size[0], a.size[0])
        h = min(b.size[1], a.size[1])
        b = b.crop((0, 0, w, h))
        a = a.crop((0, 0, w, h))
        diff = ImageChops.difference(b, a)
        r, g, bl = diff.split()
        worst = ImageChops.lighter(ImageChops.lighter(r, g), bl)
        binary = worst.point(lambda v: 255 if v > 0 else 0)
        for name, (x0, y0, x1, y1) in REGIONS_DIP.items():
            box = (int(x0 * scale), int(y0 * scale), int(x1 * scale), int(y1 * scale))
            box = (max(0, box[0]), max(0, box[1]), min(w, box[2]), min(h, box[3]))
            if box[2] <= box[0] or box[3] <= box[1]:
                continue
            part = binary.crop(box)
            total = part.size[0] * part.size[1]
            data = part.get_flattened_data() if hasattr(part, "get_flattened_data") else part.getdata()
            bad = sum(1 for v in data if v)
            regions[name] = {"box_px": list(box), "pixels": total, "mismatch": bad, "ratio": round(bad / total, 4)}

    summary = {
        "row": row.name,
        "tolerance_dip": TOLERANCE_DIP,
        "scale": scale,
        "registration": {
            "native_origin": {"x": nox, "y": noy, "detected_by": norigin_how},
            "mygo_origin": {"x": mox, "y": moy, "detected_by": morigin_how},
            "native_caption_bottom": {"y": round(ncaption, 2), "detected_by": ncaption_how},
            "mygo_caption_bottom": {"y": round(mcaption, 2), "detected_by": mcaption_how},
            "caption_offset_dip": round(mcaption - ncaption, 2),
            "note": (
                "caption-relative subtracts each side's own caption bottom, so a control placed "
                "a few DIP lower inside a same-height band does not smear into every page delta; "
                "caption_offset_dip records how much that registration absorbed."
            ),
        },
        "native_controls": sum(len(v) for v in index(native_nodes, "absolute", "native").values()),
        "mygo_controls": sum(len(v) for v in index(mygo_nodes, "absolute", "mygo").values()),
        "aliases": ALIASES,
        "absolute": results["absolute"],
        "window": results["window"],
        "caption": results["caption"],
        "window_outside_dy_median": dy_median_outside,
        "regions": regions,
    }
    (row / out_name).write_text(json.dumps(summary, indent=2, ensure_ascii=False), encoding="utf-8")
    brief = {
        "row": summary["row"],
        "registration": summary["registration"],
        "absolute": {k: summary["absolute"][k] for k in ("matched", "within_1dip", "outside_1dip", "missing_native_only", "extra_mygo_only")},
        "window": {k: summary["window"][k] for k in ("matched", "within_1dip", "outside_1dip", "missing_native_only", "extra_mygo_only", "alias_matches", "normalized_matches")},
        "caption": {k: summary["caption"][k] for k in ("matched", "within_1dip", "outside_1dip", "missing_native_only", "extra_mygo_only")},
    }
    print(json.dumps(brief, indent=2, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
