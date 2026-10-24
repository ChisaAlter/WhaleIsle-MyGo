#!/usr/bin/env python3
"""AA-dilated acceptance score for one acceptance row.

Two independent text renderers cannot agree byte-for-byte on glyph antialiasing,
so a per-region pixel threshold below ~0.3 % measures AA, not layout. This script
therefore reports three numbers for the same pair:

  raw            every pixel, threshold 0 / Δ>8 / Δ>16 (the contract's literal form)
  caption-masked the registered mask.json rectangles removed (injection-only band)
  AA-excluded    additionally remove every pixel that is "text ink" in EITHER image,
                 dilated by ~1.5-2 px, so glyph edges stop counting while a shifted
                 box, a wrong surface or a missing element still does

The ink mask is a local-contrast test (3x3 max-minus-min per channel in each image,
max over channels), then dilated. Coverage is reported so the reviewer can see how
much of the frame the exclusion actually removed.

Usage:
  python scripts/score-aa.py <row-dir> [--ink 10] [--dilate 2] [--out score-aa.json]
"""

from __future__ import annotations

import json
import pathlib
import sys

from PIL import Image, ImageChops, ImageFilter, ImageStat

REGIONS_DIP = {
    "sidebar": (0, 0, 285, 860),
    "header": (285, 0, 1440, 60),
    "hero": (285, 60, 1440, 400),
    "composer": (285, 400, 1440, 620),
    "footer": (0, 800, 285, 860),
}
SCALE = 1.5
THRESHOLDS = (0, 8, 16)


def worst_channel(img: Image.Image) -> Image.Image:
    r, g, b = img.split()
    return ImageChops.lighter(ImageChops.lighter(r, g), b)


def local_contrast(img: Image.Image) -> Image.Image:
    """Max over channels of the 3x3 local range: high where glyph edges live."""
    out = None
    for band in img.split():
        rng = ImageChops.subtract(band.filter(ImageFilter.MaxFilter(3)), band.filter(ImageFilter.MinFilter(3)))
        out = rng if out is None else ImageChops.lighter(out, rng)
    return out


def load_registered_mask(row: pathlib.Path, size: tuple[int, int]) -> Image.Image | None:
    path = row / "mask.json"
    if not path.exists():
        return None
    data = json.loads(path.read_text(encoding="utf-8-sig"))
    items = data.get("mask", [])
    if not items:
        return None
    mask = Image.new("L", size, 0)
    for item in items:
        rect = item.get("rect", item)
        x, y = int(rect["x"]), int(rect["y"])
        w, h = int(rect["w"]), int(rect["h"])
        mask.paste(255, (x, y, x + w, y + h))
    return mask


def stats(values: list[int], exclude: set[int]) -> dict:
    total = len(values)
    kept = [v for i, v in enumerate(values) if i not in exclude] if exclude else values
    out = {"pixels": total, "kept": len(kept), "mean_delta": round(sum(kept) / len(kept), 3) if kept else 0.0}
    for t in THRESHOLDS:
        n = sum(1 for v in kept if v > t)
        out[f"gt{t}"] = n
        # two denominators, both reported: the kept set (what the mode scores) and the
        # whole frame (so a mode that only shrinks the denominator cannot look better)
        out[f"ratio_gt{t}"] = round(n / len(kept), 6) if kept else 0.0
        out[f"ratio_frame_gt{t}"] = round(n / total, 6) if total else 0.0
    return out


def region_stats(worst: Image.Image, excluded: dict[str, set[int]], box) -> dict:
    part = worst.crop(box)
    values = list(part.getdata())
    row = {}
    for mode, ex in excluded.items():
        row[mode] = stats(values, ex.get(box, set()))
    return row


def main() -> int:
    row = pathlib.Path(sys.argv[1])
    ink = 10
    dilate = 2
    if "--ink" in sys.argv:
        ink = int(sys.argv[sys.argv.index("--ink") + 1])
    if "--dilate" in sys.argv:
        dilate = int(sys.argv[sys.argv.index("--dilate") + 1])
    out_name = "score-aa.json"
    if "--out" in sys.argv:
        out_name = sys.argv[sys.argv.index("--out") + 1]

    base = Image.open(row / "baseline.png").convert("RGB")
    actual = Image.open(row / "actual.png").convert("RGB")
    w = min(base.size[0], actual.size[0])
    h = min(base.size[1], actual.size[1])
    base, actual = base.crop((0, 0, w, h)), actual.crop((0, 0, w, h))
    size = (w, h)
    worst = worst_channel(ImageChops.difference(base, actual))

    # ink from either side, then dilate so a 1-px shift of a glyph edge stays covered
    ink_mask = ImageChops.lighter(local_contrast(base), local_contrast(actual)).point(
        lambda v: 255 if v > ink else 0
    )
    for _ in range(dilate):
        ink_mask = ink_mask.filter(ImageFilter.MaxFilter(3))
    caption = load_registered_mask(row, size)

    ink_idx = [i for i, v in enumerate(ink_mask.getdata()) if v]
    cap_idx = [i for i, v in enumerate(caption.getdata()) if v] if caption is not None else []
    both = sorted(set(ink_idx) | set(cap_idx))

    def idx_to_boxes(index: list[int]) -> dict:
        """Per region: the excluded index set, translated to region-local indices."""
        out = {}
        pts = {(i % w, i // w) for i in index}
        for name, (x0, y0, x1, y1) in REGIONS_DIP.items():
            bx0, by0 = int(x0 * SCALE), int(y0 * SCALE)
            bx1, by1 = min(w, int(x1 * SCALE)), min(h, int(y1 * SCALE))
            bw = bx1 - bx0
            local = set()
            for (px, py) in pts:
                if bx0 <= px < bx1 and by0 <= py < by1:
                    local.add((py - by0) * bw + (px - bx0))
            out[name] = local
        return out

    ink_boxes = idx_to_boxes(ink_idx)
    caption_boxes = idx_to_boxes(cap_idx)
    full_ink = set(range(w * h)) if False else set(ink_idx)

    modes = {"raw": {}, "caption_masked": {k: v for k, v in caption_boxes.items()}, "aa_excluded": {}}

    def region_exclusions(mode: str) -> dict:
        if mode == "raw":
            return {}
        if mode == "caption_masked":
            return {name: set() for name in REGIONS_DIP} | caption_boxes
        # aa_excluded
        out = {}
        for name in REGIONS_DIP:
            out[name] = set(caption_boxes.get(name, set())) | set(ink_boxes.get(name, set()))
        return out

    report: dict = {
        "row": row.name,
        "size": {"w": w, "h": h},
        "ink_threshold": ink,
        "dilate_px": dilate,
        "ink_coverage": round(len(ink_idx) / (w * h), 4),
        "caption_coverage": round(len(cap_idx) / (w * h), 4),
        "absolute": {},
        "regions": {},
    }
    # absolute: no region offset, so use a direct mask over whole image
    report["absolute"]["raw"] = stats(list(worst.getdata()), set())
    report["absolute"]["caption_masked"] = stats(list(worst.getdata()), set(cap_idx))
    report["absolute"]["aa_excluded"] = stats(list(worst.getdata()), set(both))

    for name, (x0, y0, x1, y1) in REGIONS_DIP.items():
        box = (int(x0 * SCALE), int(y0 * SCALE), min(w, int(x1 * SCALE)), min(h, int(y1 * SCALE)))
        part = worst.crop(box)
        values = list(part.getdata())
        region_row = {}
        region_row["raw"] = stats(values, set())
        region_row["caption_masked"] = stats(values, caption_boxes.get(name, set()))
        region_row["aa_excluded"] = stats(values, set(caption_boxes.get(name, set())) | set(ink_boxes.get(name, set())))
        report["regions"][name] = region_row

    (row / out_name).write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    print(json.dumps({"absolute": report["absolute"], "ink_coverage": report["ink_coverage"]}, ensure_ascii=False, indent=2))
    for name, row_ in report["regions"].items():
        print(name, {mode: {k: row_[mode][k] for k in ("ratio_gt0", "ratio_gt8", "ratio_gt16", "mean_delta")} for mode in row_})
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
