// Summarises a frame trace by interaction, and the open frame on its own.
//
//	node scripts/open-frame-report.mjs .devdata/frame-probe/open-300.csv
//
// A frame is named after the last mark the probe set before it, so the frames
// marked open are the first frame of each run: the one where the store
// projects the view and the chat projects its messages.
import { readFileSync } from "node:fs";

const file = process.argv[2];
if (!file) {
  console.error("usage: node scripts/open-frame-report.mjs <trace.csv>");
  process.exit(2);
}

const lines = readFileSync(file, "utf8").split(/\r?\n/).filter(Boolean);
const header = lines[0].split(",");
const column = Object.fromEntries(header.map((name, index) => [name, index]));
const us = (row, name) => Number(row[column[name]]) / 1000;
const kib = (row, name) => Number(row[column[name]]) / 1024;

const frames = [];
let mark = "";
for (const line of lines.slice(1)) {
  const row = line.split(",");
  if (row[0] === "mark") {
    mark = row[column.name];
    continue;
  }
  if (row[0] !== "frame") continue;
  frames.push({
    mark: mark || "(unmarked)",
    duration: us(row, "dur_ns"),
    build: us(row, "build_ns"),
    layout: us(row, "layout_ns"),
    paint: us(row, "paint_ns"),
    present: us(row, "present_ns"),
    alloc: kib(row, "alloc_bytes"),
    gc: Number(row[column.gc_cycles]),
  });
}

const percentile = (values, p) => {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.min(sorted.length - 1, Math.ceil(sorted.length * p) - 1)] ?? 0;
};
const fmt = (n) => n.toFixed(n >= 100 ? 0 : 1);

const open = frames.filter((f) => f.mark.endsWith("open"));
const rest = frames.filter((f) => !f.mark.endsWith("open"));

console.log(`${file}: ${frames.length} frames`);
for (const [index, frame] of open.entries()) {
  console.log(
    `  open ${index}: ${fmt(frame.duration)}us` +
      ` (build ${fmt(frame.build)}, layout ${fmt(frame.layout)},` +
      ` paint ${fmt(frame.paint)}, present ${fmt(frame.present)})` +
      ` ${fmt(frame.alloc)}KiB, gc ${frame.gc}`,
  );
}
if (rest.length) {
  const durations = rest.map((f) => f.duration);
  console.log(
    `  other ${rest.length} frames: p50 ${fmt(percentile(durations, 0.5))}us,` +
      ` p95 ${fmt(percentile(durations, 0.95))}, p99 ${fmt(percentile(durations, 0.99))},` +
      ` max ${fmt(Math.max(...durations))},` +
      ` over 6.06ms ${durations.filter((d) => d > 6060).length},` +
      ` alloc p95 ${fmt(percentile(rest.map((f) => f.alloc), 0.95))}KiB`,
  );
}

const byMark = new Map();
for (const frame of rest) {
  if (!byMark.has(frame.mark)) byMark.set(frame.mark, []);
  byMark.get(frame.mark).push(frame);
}
for (const [name, group] of byMark) {
  const durations = group.map((f) => f.duration);
  console.log(
    `  ${name}: ${group.length} frames, p95 ${fmt(percentile(durations, 0.95))}us,` +
      ` max ${fmt(Math.max(...durations))}, alloc max ${fmt(Math.max(...group.map((f) => f.alloc)))}KiB`,
  );
}
