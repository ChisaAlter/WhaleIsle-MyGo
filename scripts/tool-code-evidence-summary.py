"""Compare retained original and native run_code geometry without a rerun."""
import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / "docs/review-20261022/evidence"
PREFIX = "tool-code-pass260-reference-2026-10-24T01-20-45-073Z-113d5267"


def load(path):
    return json.loads(path.read_text(encoding="utf-8-sig"))


def main():
    original = load(EVIDENCE / (PREFIX + ".json"))
    baseline = original["measurements"]["geometry"]["final"][0]
    branches = {row["id"]: row for row in original["measurements"]["inputBranches"]["final"]}
    native = load(EVIDENCE / "tool-code-pass260-native-software-short-dark-false.json")["boxes"]
    long_native = load(EVIDENCE / "tool-code-pass260-native-software-independent-scroll.json")["boxes"]
    checks = []

    def compare(name, reference, value):
        checks.append({"name": name, "original": reference, "native": value,
                       "pass": abs(reference - value) <= 0.03})

    compare("code card width in600 row", baseline["block"]["box"]["w"], native["Tool code card"]["W"])
    compare("code left indent", baseline["block"]["box"]["x"]-baseline["viewport"]["box"]["x"], native["Tool code card"]["X"]-native["Tool program"]["X"])
    compare("single-line code card outer height", branches["invalid"]["block"]["box"]["h"], native["Tool code card"]["H"])
    compare("single-line scroll owner including margins", branches["invalid"]["viewport"]["box"]["h"], native["Tool program"]["H"])
    button = baseline["buttons"][1]
    native_button = native.get("Copy", native.get("Copied"))
    compare("toolbar button width", button["box"]["w"], native_button["W"])
    compare("toolbar button height", button["box"]["h"], native_button["H"])
    compare("toolbar top action inset", button["box"]["y"]-baseline["block"]["box"]["y"], native_button["Y"]-native["Tool code card"]["Y"])
    compare("toolbar right action inset", baseline["block"]["box"]["x"]+baseline["block"]["box"]["w"]-button["box"]["x"]-button["box"]["w"], native["Tool code card"]["X"]+native["Tool code card"]["W"]-native_button["X"]-native_button["W"])
    compare("toolbar language inset", baseline["language"]["box"]["x"]-baseline["block"]["box"]["x"], native["typescript"]["X"]-native["Tool code card"]["X"])
    compare("toolbar language line height", baseline["language"]["box"]["h"], native["typescript"]["H"])
    compare("output one-line section height", branches["empty-raw"]["io"]["section"]["box"]["h"], native["OUT"]["H"])
    scroll = original["measurements"]["scroll"]["final"][0]
    compare("long program scroll owner cap", scroll["viewport"]["box"]["h"], long_native["Tool program"]["H"])
    compare("long output independent cap", scroll["io"]["section"]["box"]["h"], long_native["OUT"]["H"])
    result = {"scope": "Dimensions only from original installed-CSS manual-DOM DPR1 and native software DPR1. Text, fonts, highlighters and fixture offsets differ; no pixel or full-app parity claim.",
              "originalAcceptance": original["summary"], "checks": checks,
              "passed": sum(item["pass"] for item in checks), "total": len(checks),
              "retainedFailure": "Original wrap check expected pre outer45 but measured50 due installed5px horizontal scrollbar. Original acceptance remains failed55/56.",
              "knownNativeDifference": "MyGo overlays scrollbars rather than reserving the original browser5px horizontal scrollbar layout. This comparison excludes unwrapped pre outer height."}
    target = EVIDENCE / "tool-code-pass260-geometry-comparison.json"
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n", encoding="utf-8")
    print(f"Geometry comparison: {result['passed']}/{result['total']} dimensions match")
    if result["passed"] != result["total"]:
        return 1
    report = ROOT / "docs/review-20261022/round116-run-code-body.md"
    if report.exists():
        links = re.findall(r"\]\(([^)]+)\)", report.read_text(encoding="utf-8"))
        missing = []
        for raw in links:
            raw = raw.strip("<>").split("#", 1)[0]
            if not raw or "://" in raw:
                continue
            path = pathlib.Path(raw)
            if not path.is_absolute():
                path = report.parent / path
            if not path.exists():
                missing.append(raw)
        print(f"Report links: {len(links)-len(missing)}/{len(links)} targets exist")
        if missing:
            print("Missing links:", missing)
            return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
