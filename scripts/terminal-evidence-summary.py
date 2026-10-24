"""Build a reproducible comparison from retained Round115 measurements."""
import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / "docs/review-20261022/evidence"
REFERENCE = EVIDENCE / "terminal-pass259-reference-2026-10-24T00-45-49-305Z-2fa10dee.json"


def load(path):
    return json.loads(path.read_text(encoding="utf-8-sig"))


def main():
    reference = load(REFERENCE)
    cases = {row["id"]: row for row in reference["measurements"]["initial"]}
    checks = []

    def compare(name, original, native):
        checks.append({"name": name, "original": original, "native": native,
                       "pass": abs(original - native) <= 0.03})

    running = load(EVIDENCE / "terminal-pass259-native-software-running.json")["boxes"]
    empty = load(EVIDENCE / "terminal-pass259-native-software-empty.json")["boxes"]
    failure = load(EVIDENCE / "terminal-pass259-native-software-failure.json")["boxes"]
    scroll = load(EVIDENCE / "terminal-pass259-native-software-independent-scroll.json")["boxes"]
    badge = load(EVIDENCE / "terminal-pass259-native-software-exit-badge.json")["boxes"]
    compare("row width600 -> terminal width", cases["running"]["block"]["box"]["w"], running["Terminal output"]["W"])
    compare("running banner-only card height", cases["running"]["block"]["box"]["h"], running["Terminal output"]["H"])
    compare("settled empty card height", cases["empty"]["block"]["box"]["h"], empty["Terminal output"]["H"])
    compare("empty section outer height", cases["empty"]["empty"]["box"]["h"], empty["No output"]["H"])
    compare("empty section own gutter", cases["empty"]["empty"]["box"]["x"]-cases["empty"]["block"]["box"]["x"], empty["No output"]["X"]-empty["Terminal output"]["X"])
    compare("single-line failure card height", cases["nonzero"]["block"]["box"]["h"], failure["Terminal output"]["H"])
    compare("single-line output outer height", cases["nonzero"]["output"]["box"]["h"], failure["Output"]["H"])
    compare("neutral row exit badge height", cases["nonzero"]["badge"]["box"]["h"], badge["exit code 7"]["H"])
    compare("long header including divider", cases["independent"]["header"]["box"]["h"], scroll["Terminal command banner"]["H"]+1)
    compare("long output content-box cap", cases["independent"]["output"]["box"]["h"], scroll["Output"]["H"])
    compare("long terminal card height", cases["independent"]["block"]["box"]["h"], scroll["Terminal output"]["H"])

    result = {
        "scope": "Dimensions only from retained original manual-DOM Chrome DPR1 and MyGo software DPR1 JSON; different fixture text/fonts. Not pixel identity or a full-app acceptance result.",
        "originalAcceptance": reference["summary"],
        "referencePath": str(REFERENCE.relative_to(ROOT)),
        "checks": checks,
        "passed": sum(row["pass"] for row in checks),
        "total": len(checks),
        "limitations": ["Original browser acceptance retains one failed extra Enter-on-copy assertion.",
                        "The header divider is a separate native element; add its measured1DIP at DPR1 for comparison.",
                        "The originals use output font11/16 with18px minimum line boxes; native joined selectable output uses fixed18px line height.",
                        "Original superellipse corner shaping and exact spinner arc easing are not certified by dimension checks."]
    }
    (EVIDENCE / "terminal-pass259-geometry-comparison.json").write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n", encoding="utf-8")
    print(f"Geometry comparison: {result['passed']}/{result['total']} dimensions match")
    if result["passed"] != result["total"]:
        return 1

    report = ROOT / "docs/review-20261022/round115-terminal-cards.md"
    if report.exists():
        missing = []
        links = re.findall(r"\]\(([^)]+)\)", report.read_text(encoding="utf-8"))
        for target in links:
            target = target.strip("<>").split("#", 1)[0]
            if not target or "://" in target:
                continue
            path = pathlib.Path(target)
            if not path.is_absolute():
                path = report.parent / path
            if not path.exists():
                missing.append(target)
        print(f"Report links: {len(links)-len(missing)}/{len(links)} targets exist")
        if missing:
            print("Missing:", missing)
            return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
