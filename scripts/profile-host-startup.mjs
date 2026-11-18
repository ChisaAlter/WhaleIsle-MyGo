// Opt-in Node preload for one owned startup investigation. Start with
// NODE_OPTIONS=--import=<file URL> and WHALEISLE_HOST_STARTUP_PROFILE=<path>.
// The output contains CPU stack locations and file counts, never file contents.
import inspector from 'node:inspector';
import fs from 'node:fs';
import { syncBuiltinESMExports } from 'node:module';

const output = process.env.WHALEISLE_HOST_STARTUP_PROFILE;
const entry = (process.argv[1] || '').replaceAll('\\', '/');
if (output && entry.endsWith('/apps/cli/lib/bin.js')) {
  const read = fs.readFileSync;
  const write = fs.writeFileSync;
  const started = performance.now();
  const reads = new Map();
  fs.readFileSync = function (path, ...args) {
    const before = performance.now();
    const value = Reflect.apply(read, this, [path, ...args]);
    const name = String(path).replaceAll('\\', '/');
    let row = reads.get(name);
    if (!row) reads.set(name, row = { path: name, calls: 0, milliseconds: 0, bytes: 0 });
    row.calls++;
    row.milliseconds += performance.now() - before;
    row.bytes += typeof value === 'string' ? Buffer.byteLength(value) : value.byteLength;
    return value;
  };
  syncBuiltinESMExports();
  const session = new inspector.Session();
  session.connect();
  const post = (method, params) => new Promise((resolve, reject) => session.post(method, params, (error, result) => error ? reject(error) : resolve(result)));
  await post('Profiler.enable');
  await post('Profiler.setSamplingInterval', { interval: 1000 });
  await post('Profiler.start');
  let stopped = false;
  let timer;
  let workspaceTimer;
  const marks = [];
  const stop = async (reason) => {
    if (stopped) return;
    stopped = true;
    clearTimeout(timer);
    clearInterval(workspaceTimer);
    const elapsedMs = performance.now() - started;
    const { profile } = await post('Profiler.stop');
    Reflect.apply(write, fs, [output, JSON.stringify(profile)]);
    Reflect.apply(write, fs, [output + '.reads.json', JSON.stringify({ reason, elapsedMs, marks, reads: [...reads.values()].sort((a, b) => b.milliseconds - a.milliseconds) }, null, 2)]);
    session.disconnect();
  };
  const stdoutWrite = process.stdout.write;
  process.stdout.write = function (chunk, ...args) {
    const result = Reflect.apply(stdoutWrite, this, [chunk, ...args]);
    if (String(chunk).includes('dsh web: http://')) {
      marks.push({ name: 'readyline', elapsedMs: performance.now() - started });
      if (process.env.WHALEISLE_HOST_PROFILE_TO_WORKSPACE === '1' && process.env.WHALEISLE_STARTUP_TRACE) {
        workspaceTimer = setInterval(() => {
          try {
            const trace = JSON.parse(Reflect.apply(read, fs, [process.env.WHALEISLE_STARTUP_TRACE, 'utf8']));
            if (trace.events.some(event => event.stage === 'workspace.ready')) void stop('workspace-ready');
          } catch { }
        }, 100);
        workspaceTimer.unref();
      } else void stop('readyline');
    }
    return result;
  };
  timer = setTimeout(() => { void stop('startup-timeout'); }, 60000);
  timer.unref();
}
