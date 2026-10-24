#!/usr/bin/env python3
"""Tolerance-based scoring for one acceptance pair.

The contract's threshold 0 counts a 1/255 base difference, which a composition
placeholder can produce even when every control is right. This report therefore
scores the same pair several ways and labels what each proves:

  delta>0    exact bytes (the contract's literal threshold)
  delta>8    ignores a uniform base/wash offset up to 8 levels, so a 9-level base
             delta (MyGo 21,21,23 vs native 12,12,12) falls under it and the
             remaining mismatches are text ink, antialiasing and layout edges
  delta>16   only strong disagreements: wrong glyphs, wrong surfaces, displacement
  wash-mask  acceptance-diff.py --mask-mode auto (low-frequency wash removed)

Per-region numbers are reported for every mode plus the mean absolute delta.

Usage: python tmp/score-thresholds.py <row-dir> [--out score-tolerance.json]
"""

from __future__ import annotations

import json
import pathlib
import sys

from PIL import Image, ImageChops, ImageStat

REGIONS_DIP = {
    "sidebar": (0, 0, 285, 860),
    "header": (285, 0, 1440, 60),
    "hero": (285, 60, 1440, 400),
    "composer": (285, 400, 1440, 620),
    "footer": (0, 800, 285, 860),
}
SCALE = 1.5
THRESHOLDS = (0, 8, 16)


def worst_channel(diff: Image.Image) -> Image.Image:
    r, g, b = diff.split()
    return ImageChops.lighter(ImageChops.lighter(r, g), b)


def region_stats(worst: Image.Image, box) -> dict:
    part = worst.crop(box)
    total = part.size[0] * part.size[1]
    if total == 0:
        return {}
    mean = ImageStat.Stat(part).mean[0]
    out = {"pixels": total, "mean_delta": round(mean, 2)}
    data = list(part.getdata())
    for t in THRESHOLDS:
        n = sum(1 for v in data if v > t)
        out[f"gt{t}"] = n
        out[f"ratio_gt{t}"] = round(n / total, 6)
    return out


def main() -> int:
    row = pathlib.Path(sys.argv[1])
    out_name = "score-tolerance.json"
    if "--out" in sys.argv:
        out_name = sys.argv[sys.argv.index("--out") + 1]
    base = Image.open(row / "baseline.png").convert("RGB")
    actual = Image.open(row / "actual.png").convert("RGB")
    w = min(base.size[0], actual.size[0])
    h = min(base.size[1], actual.size[1])
    base, actual = base.crop((0, 0, w, h)), actual.crop((0, 0, w, h))
    diff = ImageChops.difference(base, actual)
    worst = worst_channel(diff)

    report: dict = {"size": {"w": w, "h": h}, "absolute": {}}
    report["absolute"] = region_stats(worst, (0, 0, w, h))
    report["regions"] = {}
    for name, (x0, y0, x1, y1) in REGIONS_DIP.items():
        box = (
            int(x0 * SCALE),
            int(y0 * SCALE),
            min(w, int(x1 * SCALE)),
            min(h, int(y1 * SCALE)),
        )
        report["regions"][name] = region_stats(worst, box)

    wash_path = row / "diff-wash.json"
    if wash_path.exists():
        wash = json.loads(wash_path.read_text(encoding="utf-8-sig"))
        report["wash_mask"] = {
            "masked_pixels": wash.get("masked_pixels"),
            "masked_ratio": wash.get("masked_ratio"),
            "unmasked_pixels": wash.get("unmasked_mismatch_pixels"),
            "unmasked_ratio": wash.get("unmasked_mismatch_ratio"),
        }
    pathlib.Path(row / out_name).write_text(
        json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    print(json.dumps({"absolute": report["absolute"], "wash_mask": report.get("wash_mask")}, ensure_ascii=False, indent=2))
    for name, st in report["regions"].items():
        print(name, st)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
