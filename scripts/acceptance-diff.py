#!/usr/bin/env python3
"""Pixel diff for one acceptance row (contract docs/acceptance-ui-parity.md).

Loads baseline.png (native Electron) and actual.png (MyGo), honours mask.json
regions, and writes diff.png (mismatches highlighted) plus a JSON summary with
the mismatch count/ratio and the per-channel maximum delta.

A pixel counts as a mismatch when ANY channel differs by more than the threshold,
which the contract fixes at 0 (exact equality). Masked regions are excluded from
the counts but still painted grey in diff.png so a reviewer can see what was
excluded.

Wallpaper / mica masking
------------------------
The two clients do not paint the same background material: native shows the
user's wallpaper through a translucent mica sidebar and hero, MyGo paints its own
gradient. Those differences are broad and slowly varying, while a misplaced
control produces thin, edge-rich disagreement. ``--mask-mode auto`` therefore
masks *low-frequency wash* instead of guessing geometry:

  1. D = per-pixel max channel |baseline - actual|.
  2. R = local range of D over a 5x5 window (MaxFilter - MinFilter).
  3. M = local mean of D over the same window (BoxBlur).
  4. wash = D > threshold AND R <= WASH_EDGE AND M >= WASH_MEAN.

A control displacement is not masked by this test: its disagreement is an edge,
so R is large (tens of levels). A wallpaper gradient has R of only a few levels
even when its absolute difference is large, so it is masked. The method is
conservative: WASH_EDGE is 6 levels out of 255.

``--mask-mode registered`` keeps the contract's behaviour (mask.json rectangles
only), ``both`` unions them. ``--out`` names the summary file. The mask itself is
written back into the summary as ``wash_blocks`` (32 px blocks where the wash
covered at least 90 % of the block) so a reviewer can register or challenge it
without re-deriving the image.

Usage: python scripts/acceptance-diff.py <row-dir> [--threshold N]
           [--mask-mode registered|auto|both] [--out diff.json]
"""

from __future__ import annotations

import json
import pathlib
import sys

from PIL import Image, ImageChops, ImageDraw, ImageFilter

WASH_EDGE = 6
WASH_MEAN = 2
BLOCK_PX = 32

# Regions are expressed in DIP, converted with the row's DPI scale when the
# geometry dump is present (the contract fixes the capture at 144 dpi = 150 %).
REGIONS_DIP = {
    "sidebar": (0, 0, 285, 860),
    "header": (285, 0, 1440, 60),
    "hero": (285, 60, 1440, 400),
    "composer": (285, 400, 1440, 620),
    "footer": (0, 800, 285, 860),
}


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


def wash_mask(worst: Image.Image, threshold: int) -> tuple[Image.Image, dict]:
    """Mask broad low-frequency differences; return the mask and its statistics."""
    local_max = worst.filter(ImageFilter.MaxFilter(5))
    local_min = worst.filter(ImageFilter.MinFilter(5))
    local_range = ImageChops.subtract(local_max, local_min)
    local_mean = worst.filter(ImageFilter.BoxBlur(2))

    above = worst.point(lambda v: 255 if v > threshold else 0)
    smooth = local_range.point(lambda v: 255 if v <= WASH_EDGE else 0)
    broad = local_mean.point(lambda v: 255 if v >= WASH_MEAN else 0)
    wash = ImageChops.multiply(ImageChops.multiply(above, smooth), broad)
    wash = wash.point(lambda v: 255 if v > 127 else 0)
    stats = {
        "method": "low-frequency wash: max-channel D>threshold, 5x5 local range<=6, 5x5 local mean>=2",
        "wash_edge": WASH_EDGE,
        "wash_mean": WASH_MEAN,
        "wash_pixels": sum(1 for v in wash.get_flattened_data() if v),
    }
    return wash, stats


def blocks(mask: Image.Image, width: int, height: int, minimum: float = 0.9) -> list[dict]:
    """32 px blocks the mask covers at least ``minimum`` of, in client DIP."""
    out = []
    pixels = mask.load()
    for by in range(0, height, BLOCK_PX):
        for bx in range(0, width, BLOCK_PX):
            total = 0
            hit = 0
            for y in range(by, min(by + BLOCK_PX, height)):
                for x in range(bx, min(bx + BLOCK_PX, width)):
                    total += 1
                    if pixels[x, y]:
                        hit += 1
            if total and hit / total >= minimum:
                out.append({"x": bx, "y": by, "w": BLOCK_PX, "h": BLOCK_PX})
    return out


def top_blocks(image: Image.Image, width: int, height: int, limit: int = 20) -> list[dict]:
    """Blocks with the most set pixels, largest first: where the diff survives."""
    pixels = image.load()
    rows = []
    for by in range(0, height, BLOCK_PX):
        for bx in range(0, width, BLOCK_PX):
            hit = 0
            for y in range(by, min(by + BLOCK_PX, height)):
                for x in range(bx, min(bx + BLOCK_PX, width)):
                    if pixels[x, y]:
                        hit += 1
            if hit:
                rows.append({"x": bx, "y": by, "w": BLOCK_PX, "h": BLOCK_PX, "pixels": hit})
    rows.sort(key=lambda row: -row["pixels"])
    return rows[:limit]


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print(__doc__)
        return 2
    row = pathlib.Path(argv[1]).resolve()
    threshold = 0
    if "--threshold" in argv:
        threshold = int(argv[argv.index("--threshold") + 1])
    mask_mode = "registered"
    if "--mask-mode" in argv:
        mask_mode = argv[argv.index("--mask-mode") + 1]
    out_name = argv[argv.index("--out") + 1] if "--out" in argv else "diff.json"

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

    regions = load_mask(row, (width, height)) if mask_mode in ("registered", "both") else []
    mask_image = Image.new("L", (width, height), 0)
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

    stats = {}
    if mask_mode in ("auto", "both"):
        wash, stats = wash_mask(worst, threshold)
        mask_image = ImageChops.lighter(mask_image, wash)
        stats["wash_blocks"] = blocks(wash, width, height)

    inside = ImageChops.subtract(binary, mask_image)
    mismatch_pixels = sum(1 for value in inside.get_flattened_data() if value)
    masked_pixels = sum(1 for value in mask_image.get_flattened_data() if value)
    unmasked_mismatch = sum(1 for value in binary.get_flattened_data() if value)
    total = width * height

    # diff.png: grey where masked, magenta where mismatched, faint grey elsewhere.
    out = Image.new("RGB", (width, height), (255, 255, 255))
    out.paste(baseline.point(lambda value: 255 - int((255 - value) * 0.15)), (0, 0))
    draw_out = ImageDraw.Draw(out)
    for region in regions:
        draw_out.rectangle(
            [region["x"], region["y"], region["x"] + region["w"] - 1, region["y"] + region["h"] - 1],
            fill=(206, 206, 206),
        )
    if mask_mode in ("auto", "both"):
        out.paste(Image.new("RGB", (width, height), (176, 176, 176)), (0, 0), mask_image)
    highlight = Image.new("RGB", (width, height), (236, 19, 19))
    out.paste(highlight, (0, 0), inside)
    out.save(row / "diff.png")

    # Remaining mismatch by region of interest, so a reader localises what survived.
    remaining_regions = {}
    geom_path = row / "geom.json"
    if geom_path.exists():
        geom = json.loads(geom_path.read_text(encoding="utf-8-sig"))
        scale = geom["client"]["dpi"] / 96.0
        for name, (x0, y0, x1, y1) in REGIONS_DIP.items():
            box = (int(x0 * scale), int(y0 * scale), int(x1 * scale), int(y1 * scale))
            box = (max(0, box[0]), max(0, box[1]), min(width, box[2]), min(height, box[3]))
            if box[2] <= box[0] or box[3] <= box[1]:
                continue
            part = inside.crop(box)
            total_region = part.size[0] * part.size[1]
            bad = sum(1 for v in part.get_flattened_data() if v)
            remaining_regions[name] = {
                "box_px": list(box),
                "pixels": total_region,
                "mismatch": bad,
                "ratio": round(bad / total_region, 4),
            }

    summary = {
        "ok": True,
        "size": {"w": width, "h": height},
        "cropped": cropped,
        "threshold": threshold,
        "mask_mode": mask_mode,
        "pixels_total": total,
        "mismatch_pixels": mismatch_pixels,
        "mismatch_ratio": round(mismatch_pixels / total, 6),
        "unmasked_mismatch_pixels": unmasked_mismatch,
        "unmasked_mismatch_ratio": round(unmasked_mismatch / total, 6),
        "masked_pixels": masked_pixels,
        "masked_ratio": round(masked_pixels / total, 6),
        "mask_regions": len(regions),
        "wash": {k: v for k, v in stats.items() if k != "wash_blocks"},
        "wash_blocks": stats.get("wash_blocks", [])[:80],
        "remaining_blocks": top_blocks(inside, width, height),
        "channel_max_delta": channel_max_delta,
        "channel_max": channel_max,
        "remaining_regions": remaining_regions,
    }
    (row / out_name).write_text(json.dumps(summary, indent=2), encoding="utf-8")
    print(json.dumps({k: summary[k] for k in (
        "size", "mask_mode", "threshold", "pixels_total", "mismatch_pixels", "mismatch_ratio",
        "unmasked_mismatch_pixels", "unmasked_mismatch_ratio", "masked_pixels", "masked_ratio",
        "wash", "mask_regions", "channel_max_delta",
    )}))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
