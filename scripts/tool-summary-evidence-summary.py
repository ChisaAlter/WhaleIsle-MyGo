"""Compare retained installed/manual-DOM and native/software summary dimensions.

No browser or application is started and no acceptance evidence is rewritten.
"""
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / "docs/review-20261022/evidence"
PREFIX = "tool-summary-pass261-"
REFERENCE = PREFIX + "reference-2026-10-24T01-49-50-002Z-49fcaef1"


def load(name):
    return json.loads((EVIDENCE / name).read_text(encoding="utf-8-sig"))


original = load(REFERENCE + ".json")
source = load(REFERENCE + "-source.json")
scenario = next(v for v in original["measurements"]["scenarios"] if v["id"] == "short-path")
native = load(PREFIX + "native-software-mutation-rest-dark-false.json")["boxes"]
row = native["Edit src/change.go"]
path = native["src/change.go"]
suffix = native["1KB"]
totals = native["+2 -1"]
checks = []


def compare(name, installed, actual):
    checks.append({"name": name, "installed": installed, "native": actual,
                   "pass": abs(installed - actual) < .04})


compare("header height", scenario["row"]["box"]["h"], row["H"])
compare("title left inset after business icon", scenario["title"]["box"]["x"] - scenario["row"]["box"]["x"], native["Edit"]["X"] - row["X"])
compare("path link height", scenario["summary"]["box"]["h"], path["H"])
compare("content suffix left margin", scenario["suffix"]["box"]["x"] - scenario["summary"]["box"]["x"] - scenario["summary"]["box"]["w"], suffix["X"] - path["X"] - path["W"])
compare("diff totals left margin", scenario["totals"]["box"]["x"] - scenario["suffix"]["box"]["x"] - scenario["suffix"]["box"]["w"], totals["X"] - suffix["X"] - suffix["W"])
compare("diff totals vertical shift", scenario["totals"]["box"]["y"] - scenario["row"]["box"]["y"], totals["Y"] - row["Y"])
compare("content suffix line height", scenario["suffix"]["box"]["h"], suffix["H"])
flow = native["Following summary flow"]
compare("closed body flow contribution", scenario["row"]["box"]["h"], flow["Y"] - row["Y"])

result = {
    "scope": "Retained DPR1 installed-CSS/manual-DOM and native software dimensions; different text/fonts. No native underline/font/grammar/focus/motion or complete app pixel parity claim.",
    "originalAcceptance": original["summary"],
    "sourceAlgorithmCasesPassed": sum(v["pass"] for v in source["sourceOnly"]["diffTotals"]["table"]),
    "checks": checks,
    "nativeClippingLimit": "Find reports viewport-clipped stat height23.5 after its .5 offset; installed getBoundingClientRect reports full24. Visible glyphs fit. Full outer height is not compared.",
    "linkDecorationLimit": "Native dot spacing and baseline use native font metrics; only installed dotted1px/offset3 source and native implementation are retained, not underline pixel identity.",
}
(EVIDENCE / (PREFIX + "geometry-comparison.json")).write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(f"Retained summary dimensions: {sum(v['pass'] for v in checks)}/{len(checks)}")
assert all(v["pass"] for v in checks), result

report = ROOT / "docs/review-20261022/round117-file-summary-links.md"
if report.exists():
    links = re.findall(r"\]\((?:<([^>]+)>|([^\s)]+))\)", report.read_text(encoding="utf-8"))
    failures = []
    for pair in links:
        target = next(v for v in pair if v)
        if target.startswith(("https://", "http://", "#")):
            continue
        target = target.split("#", 1)[0]
        resolved = Path(target) if Path(target).is_absolute() else report.parent / target
        if not resolved.exists():
            failures.append(target)
    print(f"Report links: {len(links)-len(failures)}/{len(links)}")
    assert not failures, failures
