"""Compare native image geometry with the retained installed-browser run."""
from pathlib import Path
import hashlib
import json

root = Path(__file__).resolve().parents[1]
evidence = root / "docs/review-20261022/evidence"
prefix = "tool-image-pass266-reference-2026-10-24T04-52-44-248Z-778cf0b5"
browser = json.loads((evidence / (prefix + ".json")).read_text(encoding="utf-8"))
source = json.loads((evidence / (prefix + "-source.json")).read_text(encoding="utf-8"))
native = json.loads((evidence / "tool-image-pass266-native-preset-geometry.json").read_text(encoding="utf-8"))
checks = []

def near(name, actual, expected, tolerance=.04):
    checks.append({"name": name, "actual": actual, "expected": expected, "tolerance": tolerance, "pass": abs(actual - expected) <= tolerance})

installed = browser["measurements"]["initial"]
a, b = native["single-en"], installed["single-en"]
near("single row width", a["body"]["W"], b["root"]["box"]["w"])
near("single image width", a["scene.png, click to view original"]["W"], b["images"][0]["box"]["w"])
near("single image height", a["scene.png, click to view original"]["H"], b["images"][0]["box"]["h"])
near("image left margin", a["scene.png, click to view original"]["X"], b["imageBody"]["box"]["x"] - b["root"]["box"]["x"])
near("image top margin", a["scene.png, click to view original"]["Y"], b["imageBody"]["box"]["y"] - b["wrap"]["box"]["y"])
near("metadata path y", a["C:\\fixture\\scene.png"]["Y"], b["meta"]["lines"][0]["box"]["y"] - b["wrap"]["box"]["y"])
near("metadata path line height", a["C:\\fixture\\scene.png"]["H"], b["meta"]["lines"][0]["box"]["h"])
near("metadata measure y", a["640x360px, 12345 bytes"]["Y"], b["meta"]["lines"][1]["box"]["y"] - b["wrap"]["box"]["y"])
near("metadata measure height", a["640x360px, 12345 bytes"]["H"], b["meta"]["lines"][1]["box"]["h"])
extra = "  leading spaces  \n\tTabbed line\n\ntrailing  "
near("extra raw text height", a[extra]["H"], b["extra"]["box"]["h"])
near("extra top position", a[extra]["Y"], b["extra"]["box"]["y"] - b["wrap"]["box"]["y"])
near("complete body including margins", a["body"]["H"], b["wrap"]["box"]["h"])
a, b = native["multi5"], installed["multi5"]
near("multi row width", a["body"]["W"], b["root"]["box"]["w"])
near("multi tile width", a["scene.png, click to view original"]["W"], b["images"][0]["box"]["w"])
near("multi tile height", a["scene.png, click to view original"]["H"], b["images"][0]["box"]["h"])
near("multi metadata begins after wrap", a["640x360px, 12345 bytes"]["Y"], b["meta"]["lines"][0]["box"]["y"] - b["wrap"]["box"]["y"])
near("multi complete body", a["body"]["H"], b["wrap"]["box"]["h"])
hashes = [{"id": item["id"], "path": item["path"], "pass": hashlib.sha256(Path(item["path"]).read_bytes()).hexdigest() == item["sha256"]} for item in source["files"]]
result = {"schema": "tool-image-pass266-reconciled/v1", "browserStatus": browser["status"], "browserSummary": browser["summary"], "browserFailuresRetained": True, "geometry": {"passed": sum(x["pass"] for x in checks), "total": len(checks), "checks": checks}, "sources": {"passed": sum(x["pass"] for x in hashes), "total": len(hashes), "checks": hashes}, "limits": ["Native source images use recorded fixture rasters and the existing session loader; no Host service executed.", "Line boxes and frame dimensions do not certify text glyph or crop pixel equality.", "Retained browser inventory failure is not cleared by this independent comparison."]}
(evidence / "tool-image-pass266-reconciled.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print("Geometry", result["geometry"]["passed"], "/", len(checks), "source hashes", result["sources"]["passed"], "/", len(hashes), "browser retained", browser["status"])
if not all(x["pass"] for x in checks + hashes):
    raise SystemExit(1)
