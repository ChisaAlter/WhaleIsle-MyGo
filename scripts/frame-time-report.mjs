#!/usr/bin/env node
// Summarises a MyGo frame trace: the frames between two FrameTraceMark rows are
// one interaction, and each gets its frame intervals, its frame cost, and how the
// cost divided into build, layout, paint and present.
//
//   node scripts/frame-time-report.mjs <trace.csv> [--hz 165] [--runs] [--json out.json]
//
// The trace comes from MYGO_FRAME_TRACE, which scripts/frame-probe.ps1 sets for a
// real window and TestFrameProbe sets for the in-memory harness. A probe that
// repeats its script marks its runs r0_, r1_ and so on; those merge here.
//
// Two clocks, because one is not enough. dur_ns is the monotonic clock, which
// steps in half milliseconds on machines whose clock is that coarse, so it only
// resolves frames of a millisecond and more. cycles are the thread's, which
// resolve any frame; the cal row the trace starts with measures how many of them
// make a second, and the cost columns convert with it. --hz turns the frame
// intervals into dropped frames, which only a real window can show.
import { readFileSync, writeFileSync } from 'node:fs'

const args = process.argv.slice(2)
const tracePath = args.find((a) => !a.startsWith('--'))
if (!tracePath) {
  console.error('usage: frame-time-report.mjs <trace.csv> [--hz 165] [--runs] [--json out.json]')
  process.exit(2)
}
const flag = (name, fallback) => {
  const at = args.indexOf(`--${name}`)
  return at >= 0 && args[at + 1] ? args[at + 1] : fallback
}
const hz = Number(flag('hz', '0'))
const jsonPath = flag('json', '')
const perRun = args.includes('--runs')

const PHASES = ['build', 'layout', 'paint', 'present']
const rows = readFileSync(tracePath, 'utf8')
  .split(/\r?\n/)
  .filter((line) => line && !line.startsWith('kind,'))
  .map((line) => {
    const cell = line.split(',')
    const num = (i) => Number(cell[i]) || 0
    return {
      kind: cell[0],
      frame: num(1),
      start: num(2),
      gap: num(3),
      dur: num(4),
      build: num(5),
      layout: num(6),
      paint: num(7),
      present: num(8),
      path: cell[10],
      name: cell[11],
      cycles: num(12),
      cBuild: num(13),
      cLayout: num(14),
      cPaint: num(15),
      cPresent: num(16),
      alloc: num(18),
      gc: num(19),
    }
  })

const calibration = rows.find((row) => row.kind === 'cal' && row.cycles > 0)
let rateSource = 'measured before the first frame'
let cyclesPerSecond = calibration ? calibration.cycles : 0
if (!cyclesPerSecond) {
  let cycles = 0
  let dur = 0
  for (const row of rows) {
    if (row.kind === 'frame' && row.cycles > 0 && row.dur > 0) {
      cycles += row.cycles
      dur += row.dur
    }
  }
  cyclesPerSecond = dur > 0 ? (cycles / dur) * 1e9 : 0
  rateSource = 'inferred from the wall clock'
}
const toUs = (cycles) => (cyclesPerSecond > 0 ? (cycles / cyclesPerSecond) * 1e6 : 0)

const windows = []
let current = { name: 'before_first_mark', frames: [] }
for (const row of rows) {
  if (row.kind === 'mark') {
    if (current.frames.length) windows.push(current)
    current = { name: row.name, frames: [] }
    continue
  }
  if (row.kind === 'frame') current.frames.push(row)
}
if (current.frames.length) windows.push(current)

const quantile = (sorted, q) => (sorted.length ? sorted[Math.min(sorted.length - 1, Math.max(0, Math.ceil(q * sorted.length) - 1))] : 0)
const asc = (values) => [...values].sort((a, b) => a - b)
const us = (ns) => ns / 1000
const budget = hz > 0 ? 1e9 / hz : 0
const interaction = (name) => name.replace(/^r\d+_/, '')

function summarise(name, frames, spans = []) {
  const gaps = asc(frames.map((f) => f.gap).filter((g) => g >= 0))
  const durs = asc(frames.map((f) => f.dur))
  const costs = asc(frames.map((f) => toUs(f.cycles)))
  const first = frames[0]
  const last = frames[frames.length - 1]
  const spanNs = last.start + last.dur - first.start
  const phase = {}
  for (const phaseName of PHASES) {
    const key = 'c' + phaseName[0].toUpperCase() + phaseName.slice(1)
    const values = asc(frames.map((f) => toUs(f[key])))
    phase[phaseName] = { p50: quantile(values, 0.5), p95: quantile(values, 0.95) }
  }
  return {
    name,
    frames: frames.length,
    span_ms: spans.length ? spans.reduce((a, b) => a + b, 0) / 1e6 : us(spanNs),
    gap_p50: us(quantile(gaps, 0.5)),
    gap_p95: us(quantile(gaps, 0.95)),
    gap_p99: us(quantile(gaps, 0.99)),
    dropped: budget > 0 ? gaps.filter((g) => g > budget * 1.5).length : null,
    wall_p50: us(quantile(durs, 0.5)),
    wall_p95: us(quantile(durs, 0.95)),
    wall_p99: us(quantile(durs, 0.99)),
    cost_p50: quantile(costs, 0.5),
    cost_p95: quantile(costs, 0.95),
    cost_p99: quantile(costs, 0.99),
    over_budget: budget > 0 ? costs.filter((c) => c > budget / 1000).length : null,
    alloc_mb: frames.reduce((a, f) => a + f.alloc, 0) / 1e6,
    gc_frames: frames.filter((f) => f.gc > 0).length,
    unmeasured: frames.filter((f) => f.cycles === 0).length,
    phase,
  }
}

const merged = new Map()
for (const window of windows) {
  const key = interaction(window.name)
  if (!merged.has(key)) merged.set(key, { frames: [], spans: [] })
  const group = merged.get(key)
  group.frames.push(...window.frames)
  const last = window.frames[window.frames.length - 1]
  group.spans.push(last.start + last.dur - window.frames[0].start)
}
const report = [...merged].map(([name, group]) => summarise(name, group.frames, group.spans))
const runs = perRun ? windows.map((window) => summarise(window.name, window.frames)) : []

const fmt = (n, digits = 2) => (Number.isFinite(n) ? n.toFixed(digits) : '-')
const table = (list) => {
  console.log(`| interaction | frames | span ms | gap p50/p95/p99 ms |${hz > 0 ? ' dropped |' : ''} wall p50/p95/p99 ms |`)
  console.log(`| --- | ---: | ---: | --- |${hz > 0 ? ' ---: |' : ''} --- |`)
  for (const r of list) {
    const dropped = hz > 0 ? ` ${r.dropped} |` : ''
    console.log(
      `| ${r.name} | ${r.frames} | ${fmt(r.span_ms, 0)} | ${fmt(r.gap_p50)} / ${fmt(r.gap_p95)} / ${fmt(r.gap_p99)} |${dropped} ${fmt(r.wall_p50)} / ${fmt(r.wall_p95)} / ${fmt(r.wall_p99)} |`,
    )
  }
  console.log('')
  console.log(`| interaction | cost p50/p95/p99 us |${budget > 0 ? ' cost > budget |' : ''} alloc MB | GC frames | unmeasured | ${PHASES.map((p) => `${p} p50/p95 us`).join(' | ')} |`)
  console.log(`| --- | --- |${budget > 0 ? ' ---: |' : ''} ---: | ---: | ---: | ${PHASES.map(() => '---').join(' | ')} |`)
  for (const r of list) {
    const over = budget > 0 ? ` ${r.over_budget} |` : ''
    console.log(
      `| ${r.name} | ${fmt(r.cost_p50)} / ${fmt(r.cost_p95)} / ${fmt(r.cost_p99)} |${over} ${fmt(r.alloc_mb, 1)} | ${r.gc_frames} | ${r.unmeasured} | ${PHASES.map((p) => `${fmt(r.phase[p].p50)} / ${fmt(r.phase[p].p95)}`).join(' | ')} |`,
    )
  }
}

console.log(`trace: ${tracePath}`)
console.log(`frames: ${rows.filter((r) => r.kind === 'frame').length}   interactions: ${report.length}`)
console.log(`clock: ${fmt(cyclesPerSecond / 1e9, 2)} cycles/ns (${rateSource})`)
if (budget > 0) console.log(`budget at ${hz} Hz: ${fmt(budget / 1000)} us per frame`)
console.log('')
table(report)
if (runs.length) {
  console.log('')
  console.log('Per run:')
  console.log('')
  table(runs)
}

if (jsonPath) {
  writeFileSync(jsonPath, JSON.stringify({ trace: tracePath, hz, cyclesPerSecond, windows: report, runs }, null, 2))
  console.log(`\njson: ${jsonPath}`)
}
