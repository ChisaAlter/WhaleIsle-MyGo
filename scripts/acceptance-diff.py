#!/usr/bin/env python3
"""Pixel diff for one acceptance row (contract docs/acceptance-ui-parity.md).

Loads baseline.png (native Electron) and actual.png (MyGo), honours mask.json
regions, and writes diff.png (mismatches highlighted) plus a JSON summary with
the mismatch count/ratio and the per-channel maximum delta.

A pixel counts as a mismatch when ANY channel differs by more than the threshold,
which the contract fixes at 0 (exact equality). Masked regions are excluded from
the counts but still painted grey in diff.png so a reviewer can see what was
excluded.

Usage: python scripts/acceptance-diff.py <row-dir> [--threshold N]
"""

from __future__ import annotations

import json
import pathlib
import sys

from PIL import Image, ImageChops


def load_mask(row: pathlib.Path, size: tuple[int, int]) -> list[dict]:
    path = row / "mask.json"
    if not path.exists():
        return []
    data = json.loads(path.read_text(encoding="utf-8-sig"))
    regions = []
    for item in data.get("mask", []):
        rect = item.get("rect", item)
        regions.append({
            "x": int(rect["x"]),
            "y": int(rect["y"]),
            "w": int(rect["w"]),
            "h": int(rect["h"]),
            "reason": item.get("reason", "registered"),
        })
    return regions


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print(__doc__)
        return 2
    row = pathlib.Path(argv[1]).resolve()
    threshold = 0
    if "--threshold" in argv:
        threshold = int(argv[argv.index("--threshold") + 1])

    baseline = Image.open(row / "baseline.png").convert("RGB")
    actual = Image.open(row / "actual.png").convert("RGB")
    cropped = None
    if baseline.size != actual.size:
        # The contract allows the two clients to differ by at most 1 px; crop both
        # to the common size instead of failing, and record the crop.
        delta = (abs(baseline.size[0] - actual.size[0]), abs(baseline.size[1] - actual.size[1]))
        if max(delta) > 1:
            print(json.dumps({"ok": False, "error": "size mismatch", "baseline": baseline.size, "actual": actual.size}))
            return 1
        width = min(baseline.size[0], actual.size[0])
        height = min(baseline.size[1], actual.size[1])
        cropped = {"baseline": list(baseline.size), "actual": list(actual.size), "used": [width, height]}
        baseline = baseline.crop((0, 0, width, height))
        actual = actual.crop((0, 0, width, height))

    width, height = baseline.size
    diff = ImageChops.difference(baseline, actual)

    # Per-channel maximum delta before masking, so the summary always carries the
    # worst channel difference even when everything is masked away.
    extrema = diff.getextrema()
    channel_max = {name: extrema[i][1] for i, name in enumerate(("r", "g", "b"))}
    channel_max_delta = max(channel_max.values())

    regions = load_mask(row, (width, height))
    mask_image = Image.new("L", (width, height), 0)
    from PIL import ImageDraw
    draw = ImageDraw.Draw(mask_image)
    for region in regions:
        draw.rectangle(
            [region["x"], region["y"], region["x"] + region["w"] - 1, region["y"] + region["h"] - 1],
            fill=255,
        )

    # Mismatch mask: any channel above the threshold.
    red, green, blue = diff.split()
    worst = ImageChops.lighter(ImageChops.lighter(red, green), blue)
    if threshold > 0:
        binary = worst.point(lambda value, t=threshold: 255 if value > t else 0)
    else:
        binary = worst.point(lambda value: 255 if value > 0 else 0)

    inside = ImageChops.subtract(binary, mask_image)
    mismatch_pixels = sum(1 for value in inside.getdata() if value)
    mask_pixels = sum(1 for value in mask_image.getdata() if value)
    total = width * height

    # diff.png: grey where masked, magenta where mismatched, faint grey elsewhere.
    out = Image.new("RGB", (width, height), (255, 255, 255))
    out.paste(baseline.point(lambda value: 255 - int((255 - value) * 0.15)), (0, 0))
    for region in regions:
        draw_rect = ImageDraw.Draw(out)
        draw_rect.rectangle(
            [region["x"], region["y"], region["x"] + region["w"] - 1, region["y"] + region["h"] - 1],
            fill=(206, 206, 206),
        )
    highlight = Image.new("RGB", (width, height), (236, 19, 19))
    out.paste(highlight, (0, 0), inside)
    out.save(row / "diff.png")

    summary = {
        "ok": True,
        "size": {"w": width, "h": height},
        "cropped": cropped,
        "threshold": threshold,
        "pixels_total": total,
        "mismatch_pixels": mismatch_pixels,
        "mismatch_ratio": round(mismatch_pixels / total, 6),
        "masked_pixels": mask_pixels,
        "mask_regions": len(regions),
        "channel_max_delta": channel_max_delta,
        "channel_max": channel_max,
    }
    (row / "diff.json").write_text(json.dumps(summary, indent=2), encoding="utf-8")
    print(json.dumps(summary))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
