"""Compare retained read-card geometry without starting an application."""
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EVIDENCE = ROOT / "docs/review-20261022/evidence"
PREFIX = "tool-read-pass262-reference-2026-10-24T02-30-31-243Z-f174167e"


def load(name):
    return json.loads((EVIDENCE / name).read_text(encoding="utf-8-sig"))


def main():
    browser = load(PREFIX + ".json")
    native = load("tool-read-pass262-native-preset-geometry.json")
    source = load(PREFIX + "-source.json")
    by_id = {item["id"]: item for item in browser["measurements"]["final"]}
    checks = []

    def compare(name, original, current, tolerance=0.01):
        checks.append({"name": name, "original": original, "native": current,
                       "tolerance": tolerance, "pass": abs(original - current) <= tolerance})

    for item in native:
        name = item["case"]
        ref = by_id[name]
        rects = item["rects"]
        compare(name + " card height", ref["block"]["box"]["h"], rects["card"]["H"])
        wrap = next(button for button in ref["buttons"] if button["kind"] == "wrap")["box"]
        compare(name + " wrap width", wrap["w"], rects["wrap"]["W"])
        compare(name + " wrap height", wrap["h"], rects["wrap"]["H"])
        compare(name + " wrap top inset", wrap["y"] - ref["block"]["box"]["y"], rects["wrap"]["Y"] - rects["card"]["Y"])
        if "firstGutter" in rects:
            gutter = ref["rows"][0]["gutter"]["box"]
            compare(name + " first row top", gutter["y"] - ref["block"]["box"]["y"], rects["firstGutter"]["Y"])
            compare(name + " first row height", gutter["h"], rects["firstGutter"]["H"])
            compare(name + " gutter width", gutter["w"], rects["firstGutter"]["W"], 0.02)
    for dark in (False, True):
        for state in ("folded", "expanded"):
            value = load("tool-read-pass262-native-software-" + state + "-dark-" + str(dark).lower() + ".json")
            height = value["rects"]["Read content card"]["H"]
            original = by_id["window14"]["header"]["box"]["h"] + browser["measurements"]["fold"]["collapsedHeight" if state == "folded" else "expandedHeight"]
            compare(state + " card dark=" + str(dark), original, height)
    mismatches = [entry["path"] for entry in source["files"]
                  if hashlib.sha256(Path(entry["path"]).read_bytes()).hexdigest().lower() != entry["sha256"].lower()]
    assert not mismatches, mismatches
    result = {"scope": "Retained local ReadBlock dimensions only; excludes text pixels, baselines, scrollbar painting, motion, actual clipboard failure and Host operations.",
              "passed": sum(check["pass"] for check in checks), "total": len(checks), "checks": checks,
              "sourceHashesMatched": len(source["files"]), "sourceHashesTotal": len(source["files"])}
    (EVIDENCE / "tool-read-pass262-geometry-comparison.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print("Read geometry: " + str(result["passed"]) + "/" + str(result["total"]) + "; current source hashes " + str(len(source["files"])) + "/" + str(len(source["files"])))
    assert result["passed"] == result["total"]


if __name__ == "__main__":
    main()
