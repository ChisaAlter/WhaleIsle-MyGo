"""Compare retained installed-browser and native diff geometry without rerunning a GUI."""
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EVIDENCE = ROOT / "docs/review-20261022/evidence"
source_path = next(EVIDENCE.glob("tool-diff-pass263-reference-*-source.json"))
reference_path = source_path.with_name(source_path.name.replace("-source.json", ".json"))
source = json.loads(source_path.read_text(encoding="utf-8-sig"))
reference = json.loads(reference_path.read_text(encoding="utf-8-sig"))
source_checks = [{"id": f["id"], "pass": hashlib.sha256(Path(f["path"]).read_bytes()).hexdigest() == f["sha256"]} for f in source["files"]]
assert all(f["pass"] for f in source_checks), source_checks
assert reference["status"] == "passed", reference["status"]
assert reference["cleanup"]["pass"] and reference["sourceIntegrity"]["allUnchanged"]

checks = []
def compare(name, actual, expected):
    passed = abs(actual - expected) <= .02
    checks.append({"name": name, "native": actual, "browser": expected, "pass": passed})
    assert passed, checks[-1]

native = {r["case"]: r["rects"] for r in json.loads((EVIDENCE / "tool-diff-pass263-native-preset-geometry.json").read_text(encoding="utf-8-sig"))}
browser = {r["id"]: r for r in reference["measurements"]["final"]}
for case, counterpart in [("unchanged", "unchanged"), ("blank-added", "blank"), ("nine", "nine"), ("ten", "ten")]:
    n, b = native[case], browser[counterpart]
    compare(case + ": card width", n["card"]["W"], b["block"]["box"]["w"])
    compare(case + ": card height", n["card"]["H"], b["block"]["box"]["h"])
    compare(case + ": full row width", n["first"]["W"], b["rows"][0]["box"]["w"])
    compare(case + ": first row offset", n["first"]["Y"] - n["card"]["Y"], b["rows"][0]["box"]["y"] - b["block"]["box"]["y"])
    compare(case + ": line height", n["first"]["H"], b["rows"][0]["box"]["h"])
    for key in ["wrap", "copy"]:
        button = next(x["box"] for x in b["buttons"] if x["kind"] == key)
        compare(case + ": " + key + " width", n[key]["W"], button["w"])
        compare(case + ": " + key + " height", n[key]["H"], button["h"])
        compare(case + ": " + key + " top", n[key]["Y"] - n["card"]["Y"], button["y"] - b["block"]["box"]["y"])
        compare(case + ": " + key + " right", n["card"]["X"] + n["card"]["W"] - n[key]["X"] - n[key]["W"], b["block"]["box"]["x"] + b["block"]["box"]["w"] - button["x"] - button["w"])
for dark in [False, True]:
    n = json.loads((EVIDENCE / ("tool-diff-pass263-native-software-folded-dark-" + str(dark).lower() + ".json")).read_text(encoding="utf-8-sig"))["rects"]
    b = browser["ten"]
    compare(str(dark) + ": chat card left margin", n["Diff content card"]["X"] - n["Following summary flow"]["X"], b["block"]["box"]["x"] - b["root"]["box"]["x"])
    compare(str(dark) + ": chat folded card height", n["Diff content card"]["H"], b["block"]["box"]["h"])
    compare(str(dark) + ": fold control row", n["Expand 5 more diff lines"]["H"], n["+ created-00"]["H"])

result = {"schema": 1, "reference": reference_path.name, "scope": "Retained dimensions from actual installed-CSS manual DOM versus native software UI; excludes font rasterization, OS clipboard, physical input and motion.", "passed": len(checks), "total": len(checks), "sourceHashes": source_checks, "checks": checks}
output = EVIDENCE / "tool-diff-pass263-geometry-comparison.json"
output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(f"Diff geometry: {len(checks)}/{len(checks)}; current source hashes {sum(x['pass'] for x in source_checks)}/{len(source_checks)}")
