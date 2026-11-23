#!/usr/bin/env node
// Reads one scripts/perf-interaction.ps1 run (run.json + frames.csv) and checks
// the interaction gates of ARCHITECTURE.md section 11.
//
//   node scripts/perf-interaction-report.mjs <run dir>
//   node scripts/perf-interaction-report.mjs <before dir> <after dir>   (adds a comparison)
//
// Frames come from MYGO_FRAME_TRACE: start_unix_ns and dur_ns are wall clock,
// alloc_bytes is what the whole process allocated while the frame ran (background
// work included, which is the point: it competes with the frame). The window
// measured runs from the first step to the end of the sequence; startup frames
// are reported apart. Exit status 1 means a gate failed or a step was missing.
import { readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

export const GATES = {
  buildP95Ms: 8,
  frameMaxMs: 50,
  slowFrameShare: 0.05, // frames over 16 ms
  slowFrameMs: 16,
  frameAllocMaxBytes: 10 << 20,
  pluginPageAllocMaxBytes: 50 << 20,
}

const ms = (ns) => ns / 1e6

function percentile(values, p) {
  if (!values.length) return 0
  const sorted = [...values].sort((a, b) => a - b)
  const at = Math.min(sorted.length - 1, Math.max(0, Math.ceil((p / 100) * sorted.length) - 1))
  return sorted[at]
}

function readFrames(dir) {
  return readFileSync(join(dir, 'frames.csv'), 'utf8')
    .split(/\r?\n/)
    .filter((line) => line.startsWith('frame,'))
    .map((line) => {
      const c = line.split(',')
      const n = (i) => Number(c[i]) || 0
      return { frame: n(1), start: n(2), dur: n(4), build: n(5), layout: n(6), paint: n(7), present: n(8), passes: n(9), path: c[10], alloc: n(18), gc: n(19), pu: n(20), pd: n(21), ps: n(22), cause: c[23] || '' }
    })
}

function readRun(dir) {
  const raw = readFileSync(join(dir, 'run.json'), 'utf8').replace(/^\uFEFF/, '')
  const run = JSON.parse(raw)
  run.steps = Array.isArray(run.steps) ? run.steps : run.steps ? [run.steps] : []
  return run
}

// causeCounts tallies the cause column (MYGO_FRAME_TRACE column 24) by part:
// a frame asked for by several causes counts once for each. Older traces
// have no column and tally nothing.
function causeCounts(frames) {
  const out = {}
  for (const f of frames) for (const part of f.cause ? f.cause.split('+') : []) out[part] = (out[part] || 0) + 1
  return Object.entries(out).sort((a, b) => b[1] - a[1])
}

function phaseStats(frames) {
  const pick = (k) => frames.map((f) => ms(f[k]))
  const stat = (k) => ({ p50: percentile(pick(k), 50), p95: percentile(pick(k), 95), max: Math.max(0, ...pick(k)) })
  return { total: stat('dur'), build: stat('build'), layout: stat('layout'), paint: stat('paint'), present: stat('present'), 'present.upload': stat('pu'), 'present.draw': stat('pd'), 'present.swap': stat('ps') }
}

export function analyze(dir) {
  const run = readRun(dir)
  const frames = readFrames(dir)
  const steps = run.steps.filter((s) => s.t_ns > 0)
  const first = steps.length ? steps[0].t_ns : run.ready_ns
  const end = run.end_ns || (frames.length ? frames[frames.length - 1].start : 0)
  const inWindow = frames.filter((f) => f.start >= first && f.start <= end)
  const startup = frames.filter((f) => f.start < (run.ready_ns || first))

  const stepRows = run.steps.map((s, i) => {
    if (!s.t_ns) return { index: s.index, kind: s.kind, name: s.name, found: false }
    const next = run.steps.slice(i + 1).find((x) => x.t_ns > 0)
    const until = next ? next.t_ns : end
    const fs = frames.filter((f) => f.start + f.dur >= s.t_ns && f.start < until)
    const firstFrame = fs.find((f) => f.start + f.dur >= s.t_ns)
    let settle = firstFrame ? firstFrame.start + firstFrame.dur : 0
    for (let j = 1; j < fs.length; j++) {
      if (fs[j].start - (fs[j - 1].start + fs[j - 1].dur) >= 250e6) break
      settle = fs[j].start + fs[j].dur
    }
    return {
      index: s.index,
      kind: s.kind,
      name: s.name,
      found: s.found !== false,
      firstFrameMs: firstFrame ? ms(firstFrame.start + firstFrame.dur - s.t_ns) : null,
      settleMs: settle ? ms(settle - s.t_ns) : null,
      frames: fs.length,
      maxFrameMs: Math.max(0, ...fs.map((f) => ms(f.dur))),
      allocBytes: fs.reduce((a, f) => a + f.alloc, 0),
    }
  })

  const phases = phaseStats(inWindow)
  const slow = inWindow.filter((f) => ms(f.dur) > GATES.slowFrameMs).length
  const maxAlloc = Math.max(0, ...inWindow.map((f) => f.alloc))
  const pluginStep = stepRows.find((s) => s.name === '插件' && s.found)
  const software = inWindow.some((f) => /software|raster/i.test(f.path || ''))

  const gates = [
    { gate: 'build p95 ≤ 8 ms', value: phases.build.p95, pass: phases.build.p95 <= GATES.buildP95Ms },
    { gate: 'frame max ≤ 50 ms', value: phases.total.max, pass: phases.total.max <= GATES.frameMaxMs },
    { gate: 'frames > 16 ms ≤ 5%', value: inWindow.length ? slow / inWindow.length : 0, pass: inWindow.length > 0 && slow / inWindow.length <= GATES.slowFrameShare },
    { gate: 'alloc per frame ≤ 10 MiB', value: maxAlloc, pass: maxAlloc <= GATES.frameAllocMaxBytes },
    { gate: 'plugin page alloc ≤ 50 MiB', value: pluginStep ? pluginStep.allocBytes : null, pass: !!pluginStep && pluginStep.allocBytes <= GATES.pluginPageAllocMaxBytes },
    { gate: 'every step found', value: stepRows.filter((s) => !s.found).map((s) => s.name), pass: stepRows.every((s) => s.found) },
  ]

  return {
    dir,
    label: run.label || '',
    profile: run.profile,
    readyMs: run.ready_ns && run.launch_ns ? ms(run.ready_ns - run.launch_ns) : null,
    renderer: software ? 'software' : 'gpu',
    startupFrames: startup.length,
    startupFps: run.ready_ns && run.launch_ns ? startup.length / ((run.ready_ns - run.launch_ns) / 1e9) : null,
    startupCauses: causeCounts(startup),
    frames: inWindow.length,
    slowFrames: slow,
    maxAllocBytes: maxAlloc,
    gcCycles: inWindow.reduce((a, f) => a + f.gc, 0),
    phases,
    steps: stepRows,
    gates,
    pass: gates.every((g) => g.pass),
  }
}

const mb = (b) => (b == null ? '-' : (b / (1 << 20)).toFixed(1) + ' MiB')
const f1 = (v) => (v == null ? '-' : Number(v).toFixed(1))

function markdown(r) {
  const lines = []
  lines.push(`# Interaction run ${r.label || ''}`.trim(), '')
  lines.push(`profile: ${r.profile} · renderer: ${r.renderer} · ready: ${f1(r.readyMs)} ms (${f1(r.startupFps)} fps while starting) · frames: ${r.frames} (${r.slowFrames} over 16 ms) · GC cycles: ${r.gcCycles}`, '')
  if (r.startupCauses.length) lines.push(`causes while starting: ${r.startupCauses.slice(0, 8).map(([k, v]) => `${k} ${v}`).join(', ')}`, '')
  lines.push('| phase | p50 ms | p95 ms | max ms |', '|---|---|---|---|')
  for (const [k, v] of Object.entries(r.phases)) lines.push(`| ${k} | ${f1(v.p50)} | ${f1(v.p95)} | ${f1(v.max)} |`)
  lines.push('', '| step | first frame ms | settle ms | frames | max frame ms | alloc |', '|---|---|---|---|---|---|')
  for (const s of r.steps) lines.push(s.found ? `| ${s.index} ${s.kind} ${s.name} | ${f1(s.firstFrameMs)} | ${f1(s.settleMs)} | ${s.frames} | ${f1(s.maxFrameMs)} | ${mb(s.allocBytes)} |` : `| ${s.index} ${s.kind} ${s.name} | MISSING | | | | |`)
  lines.push('', '| gate | value | result |', '|---|---|---|')
  for (const g of r.gates) {
    const v = typeof g.value === 'number' ? (g.gate.includes('alloc') ? mb(g.value) : g.gate.includes('%') ? (g.value * 100).toFixed(1) + '%' : f1(g.value) + ' ms') : JSON.stringify(g.value)
    lines.push(`| ${g.gate} | ${v} | ${g.pass ? 'pass' : 'FAIL'} |`)
  }
  return lines.join('\n') + '\n'
}

function comparison(a, b) {
  const lines = ['', `# ${a.label || 'before'} → ${b.label || 'after'}`, '', '| metric | before | after |', '|---|---|---|']
  const row = (name, x, y, fmt = f1) => lines.push(`| ${name} | ${fmt(x)} | ${fmt(y)} |`)
  row('ready ms', a.readyMs, b.readyMs)
  row('build p95 ms', a.phases.build.p95, b.phases.build.p95)
  row('build max ms', a.phases.build.max, b.phases.build.max)
  row('frame max ms', a.phases.total.max, b.phases.total.max)
  row('present p95 ms', a.phases.present.p95, b.phases.present.p95)
  row('slow frame share %', a.frames ? (a.slowFrames / a.frames) * 100 : 0, b.frames ? (b.slowFrames / b.frames) * 100 : 0)
  row('max alloc per frame', a.maxAllocBytes, b.maxAllocBytes, mb)
  for (const s of a.steps) {
    const t = b.steps.find((x) => x.index === s.index)
    if (t) row(`step ${s.index} ${s.name} settle ms`, s.settleMs, t.settleMs)
  }
  return lines.join('\n') + '\n'
}

if (process.argv[1] && process.argv[1].replaceAll('\\', '/').endsWith('/perf-interaction-report.mjs')) {
  const dirs = process.argv.slice(2)
  if (!dirs.length) {
    console.error('usage: perf-interaction-report.mjs <run dir> [<after run dir>]')
    process.exit(2)
  }
  const results = dirs.map(analyze)
  let md = results.map(markdown).join('\n')
  if (results.length === 2) md += comparison(results[0], results[1])
  const out = results[results.length - 1]
  writeFileSync(join(out.dir, 'summary.json'), JSON.stringify(results.length === 2 ? { before: results[0], after: results[1] } : out, null, 2))
  writeFileSync(join(out.dir, 'summary.md'), md)
  process.stdout.write(md)
  process.exit(out.pass ? 0 : 1)
}
