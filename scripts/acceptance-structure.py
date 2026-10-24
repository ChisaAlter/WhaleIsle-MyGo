#!/usr/bin/env python3
"""Structural parity report for one acceptance row.

Reads the UIA dumps produced by scripts/acceptance-capture.ps1 (geom.json) and
matches every native control to a MyGo control by (type, name). For each match it
reports the DIP rectangle delta and whether the contract's 1-DIP tolerance holds;
unmatched native controls are "missing" and unmatched MyGo controls are "extra".

Also emits per-region pixel mismatch when baseline.png/actual.png are present, so
a reviewer sees both the structural and the rendered result in one file.

Usage: python scripts/acceptance-structure.py <row-dir>
"""

from __future__ import annotations

import json
import pathlib
import sys
from PIL import Image, ImageChops

TOLERANCE_DIP = 1.0

# Regions are expressed in DIP and converted with the row's DPI scale (150 %).
REGIONS_DIP = {
    "sidebar": (0, 0, 285, 860),
    "header": (285, 0, 1440, 60),
    "hero": (285, 60, 1440, 400),
    "composer": (285, 400, 1440, 620),
    "footer": (0, 800, 285, 860),
}


def key(node: dict) -> tuple[str, str]:
    return (node.get("type", ""), node.get("name", ""))


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print(__doc__)
        return 2
    row = pathlib.Path(argv[1]).resolve()
    geom = json.loads((row / "geom.json").read_text(encoding="utf-8-sig"))
    scale = geom["client"]["dpi"] / 96.0

    def index(nodes):
        out: dict[tuple[str, str], list[dict]] = {}
        for node in nodes:
            if not node.get("name"):
                continue
            out.setdefault(key(node), []).append(node)
        return out

    nb = index(geom["native"])
    mb = index(geom["mygo"])

    matched, missing, extra, mismatched = [], [], [], []
    for k, native_nodes in nb.items():
        mine = mb.get(k)
        if not mine:
            for node in native_nodes:
                missing.append({"type": node["type"], "name": node["name"], "native": node["dip"]})
            continue
        n = native_nodes[0]
        m = mine[0]
        dx = round(m["dip"]["x"] - n["dip"]["x"], 2)
        dy = round(m["dip"]["y"] - n["dip"]["y"], 2)
        dw = round(m["dip"]["w"] - n["dip"]["w"], 2)
        dh = round(m["dip"]["h"] - n["dip"]["h"], 2)
        record = {
            "type": n["type"],
            "name": n["name"],
            "native": n["dip"],
            "mygo": m["dip"],
            "delta": {"x": dx, "y": dy, "w": dw, "h": dh},
            "within_1dip": abs(dx) <= TOLERANCE_DIP and abs(dy) <= TOLERANCE_DIP
            and abs(dw) <= TOLERANCE_DIP and abs(dh) <= TOLERANCE_DIP,
        }
        matched.append(record)
        if not record["within_1dip"]:
            mismatched.append(record)
    for k, mygo_nodes in mb.items():
        if k not in nb:
            for node in mygo_nodes:
                extra.append({"type": node["type"], "name": node["name"], "mygo": node["dip"]})

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
            bad = sum(1 for v in part.getdata() if v)
            regions[name] = {"box_px": list(box), "pixels": total, "mismatch": bad, "ratio": round(bad / total, 4)}

    summary = {
        "row": row.name,
        "tolerance_dip": TOLERANCE_DIP,
        "scale": scale,
        "native_controls": sum(len(v) for v in nb.values()),
        "mygo_controls": sum(len(v) for v in mb.values()),
        "matched": len(matched),
        "within_1dip": sum(1 for m in matched if m["within_1dip"]),
        "outside_1dip": len(mismatched),
        "missing_native_only": len(missing),
        "extra_mygo_only": len(extra),
        "regions": regions,
        "outside_tolerance": sorted(mismatched, key=lambda m: -max(abs(m["delta"]["x"]), abs(m["delta"]["y"])))[:25],
        "missing": missing[:40],
        "extra": extra[:40],
    }
    (row / "structure.json").write_text(json.dumps(summary, indent=2, ensure_ascii=False), encoding="utf-8")
    print(json.dumps({k: v for k, v in summary.items() if k not in ("outside_tolerance", "missing", "extra")}, indent=2, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
