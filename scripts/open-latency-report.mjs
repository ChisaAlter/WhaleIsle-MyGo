// Reports how long a session took to appear, from the click that opened it to
// the frame that painted it, out of a frame trace the fixture left.
//
//	node scripts/open-latency-report.mjs .devdata/frame-probe/open-trace.csv \
//	  .devdata/frame-probe/open-post.txt
//
// The probe posts the key that opens a session and stamps the wall clock in the
// sidecar file; the fixture marks the trace when its view handles that key, so
// the click's own handling and the frame after it are both measurable. Without
// the sidecar the report starts at the mark, which leaves out the input.
import { readFileSync } from "node:fs";

const [file, post] = process.argv.slice(2);
if (!file) {
  console.error("usage: node scripts/open-latency-report.mjs <trace.csv> [post.txt]");
  process.exit(2);
}

const lines = readFileSync(file, "utf8").split(/\r?\n/).filter(Boolean);
const header = lines[0].split(",");
const column = Object.fromEntries(header.map((name, index) => [name, index]));
const ns = (row, name) => Number(row[column[name]]);
const ms = (value) => (value / 1e6).toFixed(2);

const posted = post ? Number(readFileSync(post, "utf8").trim()) : null;
const fmt = (value) => (value >= 100 ? value.toFixed(0) : value.toFixed(1));

let mark = "";
let markAt = 0;
const frames = [];
for (const line of lines.slice(1)) {
  const row = line.split(",");
  if (row[0] === "mark") {
    mark = row[column.name];
    markAt = ns(row, "start_unix_ns");
    continue;
  }
  if (row[0] !== "frame") continue;
  const start = ns(row, "start_unix_ns");
  const duration = ns(row, "dur_ns");
  frames.push({
    mark,
    markAt,
    start,
    end: start + duration,
    duration,
    build: ns(row, "build_ns"),
    layout: ns(row, "layout_ns"),
    paint: ns(row, "paint_ns"),
    present: ns(row, "present_ns"),
    alloc: Number(row[column.alloc_bytes]) / 1024,
  });
}

const opened = frames.filter((frame) => frame.mark === "open_click");
console.log(`${file}: ${frames.length} frames, ${opened.length} frames after the click that opened a session`);
if (!opened.length) {
  console.error("no frame carries the open_click mark: the probe's click opened nothing");
  process.exit(1);
}

for (const [index, frame] of opened.entries()) {
  const parts =
    `build ${fmt(frame.build / 1e3)}us, layout ${fmt(frame.layout / 1e3)},` +
    ` paint ${fmt(frame.paint / 1e3)}, present ${fmt(frame.present / 1e3)}`;
  console.log(
    `  open ${index}: ${fmt(frame.duration / 1e3)}us (${parts}), ${fmt(frame.alloc)}KiB`,
  );
  console.log(`    click handled ${ms(frame.markAt - frame.start)}ms into the frame`);
  if (posted === null) continue;
  console.log(`    input to handled:   ${ms(frame.markAt - posted)}ms`);
  console.log(`    handled to painted: ${ms(frame.end - frame.markAt)}ms`);
  console.log(`    click to painted:   ${ms(frame.end - posted)}ms`);
}

const dropped = frames.filter((frame) => frame.mark === "open_drop");
if (dropped.length) {
  console.log(
    `  the empty view it opened from: ${dropped.length} frames, last ${fmt(dropped.at(-1).duration / 1e3)}us`,
  );
}
