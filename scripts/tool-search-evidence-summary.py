"""Compare retained installed SearchBlock dimensions with native software captures."""
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EVIDENCE = ROOT / "docs/review-20261022/evidence"
REFERENCE = EVIDENCE / "tool-search-pass264-reference-2026-10-24T03-36-39-942Z-bb4cca0c.json"
reference = json.loads(REFERENCE.read_text(encoding="utf-8-sig"))
source = json.loads(REFERENCE.with_name(REFERENCE.stem + "-source.json").read_text(encoding="utf-8-sig"))
source_checks = [{"id": item["id"], "pass": hashlib.sha256(Path(item["path"]).read_bytes()).hexdigest() == item["sha256"]} for item in source["files"]]
assert all(item["pass"] for item in source_checks), source_checks
assert reference["cleanup"]["pass"] and reference["sourceIntegrity"]["allUnchanged"]
# The one browser acceptance retains its two failed keyboard/focus assumptions.
# This comparison reports only the independently measured, passing dimensions.
assert reference["status"] == "failed" and reference["summary"] == {"passedCases": 9, "failedCases": 1, "passedAssertions": 44, "failedAssertions": 2}

checks = []
def compare(name, actual, expected):
    passed = abs(actual - expected) <= .02
    checks.append({"name": name, "native": actual, "browser": expected, "pass": passed})
    assert passed, checks[-1]

native = {row["case"]: row["rects"] for row in json.loads((EVIDENCE / "tool-search-pass264-native-preset-geometry.json").read_text(encoding="utf-8-sig"))}
measurement = reference["measurements"]
early = measurement["early"]["light"]["geometry"]
final = {row["id"]: row for row in measurement["final"]}
for n, b in [("empty-paths", "paths"), ("empty-matches", "matches"), ("zero-match-file", "file"), ("one-path", "oneEmptyPath")]:
    compare(n + ": height", native[n]["card"]["H"], measurement["empty"][b]["blockHeight"])
for n, b in [("paths-eight", "eight"), ("paths-nine", "nine"), ("tail-restored", "tail")]:
    compare(n + ": height", native[n]["card"]["H"], measurement["cap"][b]["blockHeight"])
short = native["short-matches"]
compare("short: card width", short["card"]["W"], early["block"]["w"])
compare("short: card height", short["card"]["H"], early["block"]["h"])
for key, b in [("src/a.go 2", early["rows"][0]["box"]), ("1: first", early["rows"][1]["box"])]:
    compare(key + ": width", short[key]["W"], b["w"])
    compare(key + ": height", short[key]["H"], b["h"])
    compare(key + ": top", short[key]["Y"] - short["card"]["Y"], b["y"] - early["block"]["y"])
compare("short: file text left padding", short["src/a.go"]["X"], final["short"]["rows"][0]["filePath"]["box"]["x"] - final["short"]["block"]["box"]["x"])
for name in ["one-path", "zero-match-file", "short-matches", "paths-eight", "paths-nine", "tail-restored"]:
    copy, card = native[name]["copy"], native[name]["card"]
    compare(name + ": copy height", copy["H"], early["copy"]["h"])
    compare(name + ": copy width", copy["W"], early["copy"]["w"])
    compare(name + ": copy top", copy["Y"] - card["Y"], early["copy"]["y"] - early["block"]["y"])
    compare(name + ": copy right padding", card["X"] + card["W"] - copy["X"] - copy["W"], early["block"]["x"] + early["block"]["w"] - early["copy"]["x"] - early["copy"]["w"])
for dark in [False, True]:
    for state in ["folded", "expanded"]:
        n = json.loads((EVIDENCE / ("tool-search-pass264-native-software-" + state + "-dark-" + str(dark).lower() + ".json")).read_text(encoding="utf-8-sig"))["rects"]
        card = n["Search results card"]
        compare(f"{state}/{dark}: card left margin", card["X"] - n["Following summary flow"]["X"], early["block"]["x"] - early["root"]["x"])
        compare(f"{state}/{dark}: height", card["H"], measurement["groups"]["before" if state == "folded" else "expanded"]["blockHeight"])
        fold = n["Expand 4 more result lines" if state == "folded" else "Collapse results"]
        compare(f"{state}/{dark}: fold height", fold["H"], measurement["groups"]["before" if state == "folded" else "expanded"]["fold"]["h"])
        compare(f"{state}/{dark}: summary line height", n["Showing 10 of 30 matches · 2 files"]["H"], 20)

result = {"schema": 1, "reference": REFERENCE.name, "referenceStatus": reference["status"], "referenceSummary": reference["summary"], "scope": "Retained measured dimensions only. Excludes the browser's failed focus-ring and Enter assertions, font rasterization, OS clipboard, physical input and motion.", "passed": len(checks), "total": len(checks), "sourceHashes": source_checks, "checks": checks}
(EVIDENCE / "tool-search-pass264-geometry-comparison.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(f"Search dimensions: {len(checks)}/{len(checks)}; current source hashes {sum(item['pass'] for item in source_checks)}/{len(source_checks)}; browser acceptance remains {reference['status']} ({reference['summary']['passedAssertions']}/46)")
