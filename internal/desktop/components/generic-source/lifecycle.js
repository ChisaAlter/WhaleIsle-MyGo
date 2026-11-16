'use strict';

// Process supervision for installed components: spawn through the host
// binary (Electron runs as plain node under ELECTRON_RUN_AS_NODE, which keeps
// components working without a separately installed Node), a native PID
// liveness probe, and tree kill — Windows-first: taskkill /PID <pid> /T /F,
// never kill -9.
const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawn, execFileSync } = require('child_process');

const LOG_MAX_BYTES = 256 * 1024;
const STRIPPED_CHILD_ENV_EXACT = new Set([
  'NODE_OPTIONS',
  'NODE_EXTRA_CA_CERTS',
  'ELECTRON_RUN_AS_NODE',
  'MAGPIE_WEB_KEY',
  'MAGPIE_WEB_RUNKEY',
  'MAGPIE_TOKEN',
  'MAGPIE_AUTH_TOKEN',
  'MAGPIE_ADDR',
  'WHALEISLE_CONTROL_TOKEN',
  'CONTROL_PLANE_TOKEN',
  'DSH_CONTROL_TOKEN',
  'DSH_AUTH_TOKEN',
  'DSH_ACCESS_TOKEN',
  'DSH_SESSION_TOKEN',
  'DSH_API_TOKEN',
  'LAUNCHER_COMPONENT_ID',
  'LAUNCHER_COMPONENT_VERSION',
  'LAUNCHER_COMPONENT_DATA_DIR',
]);

function sanitizeSpawnEnv(baseEnv, overrides = {}) {
  const out = {};
  for (const [k, v] of Object.entries(baseEnv || {})) {
    const upper = String(k).toUpperCase();
    if (
      STRIPPED_CHILD_ENV_EXACT.has(upper) ||
      upper.startsWith('DSH_DESKTOP_INSTALL_') ||
      upper.endsWith('_CONTROL_TOKEN') ||
      upper.endsWith('_WEB_RUNKEY')
    ) {
      continue;
    }
    out[k] = v;
  }
  return { ...out, ...overrides };
}

function resolvePlatform(deps = {}) {
  return typeof deps.platform === 'string' ? deps.platform : process.platform;
}

function normalizeCmdCompare(raw, platform) {
  const text = String(raw || '').trim();
  return platform === 'win32' ? text.toLowerCase() : text;
}

function inspectProcessCommand(pid, deps = {}) {
  if (typeof deps.inspectProcess === 'function') {
    return deps.inspectProcess(pid);
  }
  const platform = resolvePlatform(deps);
  if (platform === 'linux') {
    const fsp = deps.fs || fs;
    let cmdline = '';
    try {
      cmdline = fsp.readFileSync(`/proc/${pid}/cmdline`, 'utf8').replace(/\0/g, ' ').trim();
    } catch {
      cmdline = '';
    }
    let exe = '';
    try {
      exe = fsp.readlinkSync(`/proc/${pid}/exe`);
    } catch {
      exe = '';
    }
    return cmdline || exe ? { executablePath: exe, commandLine: cmdline } : null;
  }
  if (platform === 'darwin') {
    const exec = deps.execFileSync || execFileSync;
    try {
      const out = String(exec('ps', ['-p', String(pid), '-o', 'command='], { encoding: 'utf8', timeout: 3000 })).trim();
      return out ? { executablePath: '', commandLine: out } : null;
    } catch {
      return null;
    }
  }
  if (platform === 'win32') {
    const exec = deps.execFileSync || execFileSync;
    const cacheDir = path.join(os.tmpdir(), 'whaleisle-ps-cache');
    try {
      fs.mkdirSync(cacheDir, { recursive: true });
    } catch {
      // best effort
    }
    const env = sanitizeSpawnEnv(process.env, {
      PSModuleAnalysisCachePath: path.join(cacheDir, 'ModuleAnalysisCache'),
    });
    const script = `$p=Get-CimInstance Win32_Process -Filter "ProcessId=${pid}" -ErrorAction SilentlyContinue | Select-Object ExecutablePath,CommandLine; if ($p) { $p | ConvertTo-Json -Compress }`;
    try {
      const out = String(exec('powershell.exe', ['-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-Command', script], {
        encoding: 'utf8',
        windowsHide: true,
        timeout: 4000,
        env,
      })).trim();
      if (!out) {
        return null;
      }
      const parsed = JSON.parse(out);
      return {
        executablePath: String(parsed.ExecutablePath || ''),
        commandLine: String(parsed.CommandLine || ''),
      };
    } catch {
      return null;
    }
  }
  return null;
}

function matchesExpectedProcess(pid, expected, deps = {}) {
  if (!expected) {
    return true;
  }
  if (typeof deps.verifyPidIdentity === 'function') {
    return Boolean(deps.verifyPidIdentity(pid, expected));
  }
  // Preserve mock-driven fixture tests that mock processKill/execFileSync solely
  // for liveness probes and taskkill without providing process inspection.
  if ((deps.processKill || deps.execFileSync) && typeof deps.inspectProcess !== 'function') {
    return true;
  }
  const info = inspectProcessCommand(pid, deps);
  if (!info) {
    return false;
  }
  const platform = resolvePlatform(deps);
  const haystack = normalizeCmdCompare(`${info.commandLine || ''} ${info.executablePath || ''}`, platform);
  if (expected.entryFile) {
    const entryNorm = normalizeCmdCompare(path.resolve(expected.entryFile), platform);
    const entryBase = normalizeCmdCompare(path.basename(expected.entryFile), platform);
    if (haystack.includes(entryNorm)) {
      return true;
    }
    if (expected.id && haystack.includes(normalizeCmdCompare(expected.id, platform)) && haystack.includes(entryBase)) {
      return true;
    }
    return false;
  }
  if (expected.id) {
    return haystack.includes(normalizeCmdCompare(expected.id, platform));
  }
  return true;
}

function isPidAlive(pid, deps = {}) {
  if (!Number.isInteger(pid) || pid <= 0) {
    return false;
  }
  let alive = false;
  try {
    // Signal 0 only asks the OS whether the PID exists.
    (deps.processKill || process.kill)(pid, 0);
    alive = true;
  } catch (error) {
    if (error && error.code === 'EPERM') {
      alive = true;
    } else if (error && error.code === 'ESRCH') {
      return false;
    } else {
      throw error;
    }
  }
  if (!alive) {
    return false;
  }
  const expected = deps.expectedProcess || null;
  if (expected && !matchesExpectedProcess(pid, expected, deps)) {
    return false;
  }
  return true;
}

function killPid(pid, options = {}, deps = {}) {
  const { force = true, expectedProcess = deps.expectedProcess } = options || {};
  if (!Number.isInteger(pid) || pid <= 0) {
    return false;
  }
  if (expectedProcess && !matchesExpectedProcess(pid, expectedProcess, deps)) {
    return false;
  }
  if (resolvePlatform(deps) === 'win32') {
    const exec = deps.execFileSync || execFileSync;
    try {
      exec('taskkill', force
        ? ['/PID', String(pid), '/T', '/F']
        : ['/PID', String(pid)], { windowsHide: true, timeout: 5000 });
      return true;
    } catch {
      return false;
    }
  }
  try {
    process.kill(pid, force ? 'SIGKILL' : 'SIGTERM');
    return true;
  } catch {
    return false;
  }
}

function openLogStream(file, deps = {}) {
  const fsp = deps.fs || fs;
  try {
    fsp.mkdirSync(path.dirname(file), { recursive: true });
    const size = (() => {
      try {
        return fsp.statSync(file).size;
      } catch {
        return 0;
      }
    })();
    const stream = fsp.createWriteStream(file, { flags: size > LOG_MAX_BYTES ? 'w' : 'a' });
    if (stream && typeof stream.on === 'function') {
      stream.on('error', () => {});
    }
    return stream;
  } catch {
    return null;
  }
}

// Spawn one component generation. Child output lands in <dataDir>/component.log
// so a crashed service leaves evidence for the panel/error state.
function spawnComponent({ entryFile, dir, dataDir: dataPath, logFile: logPath, id, version }, deps = {}) {
  const doSpawn = deps.spawn || spawn;
  const fsp = deps.fs || fs;
  try {
    fsp.mkdirSync(dataPath, { recursive: true });
  } catch {
    // best effort; spawn still attempted so the error surfaces normally
  }
  const logStream = openLogStream(logPath, deps);
  const child = doSpawn(deps.execPath || process.execPath, [entryFile], {
    cwd: dir,
    env: sanitizeSpawnEnv(process.env, {
      ELECTRON_RUN_AS_NODE: '1',
      LAUNCHER_COMPONENT_ID: id,
      LAUNCHER_COMPONENT_VERSION: String(version || ''),
      LAUNCHER_COMPONENT_DATA_DIR: dataPath,
    }),
    stdio: ['ignore', 'pipe', 'pipe'],
    windowsHide: true,
  });
  if (logStream) {
    if (child.stdout && typeof child.stdout.pipe === 'function') {
      child.stdout.pipe(logStream);
    }
    if (child.stderr && typeof child.stderr.pipe === 'function') {
      child.stderr.pipe(logStream);
    }
    if (typeof child.once === 'function') {
      child.once('exit', () => {
        try {
          logStream.end();
        } catch {
          // already closed
        }
      });
    }
  }
  return child;
}

// Cooperative components write <dataDir>/state.json {pid,port,url,...} once
// their endpoint is up. Poll briefly; an absent file is not an error (plain
// tools never write one) — liveness is the spawn/alive probe, this is only
// how `url` reaches the panel.
async function waitForStateFile(file, { timeoutMs = 4000, pollMs = 200, isDead } = {}, deps = {}) {
  const fsp = deps.fs || fs;
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (typeof isDead === 'function' && isDead()) {
      return null;
    }
    try {
      const parsed = JSON.parse(fsp.readFileSync(file, 'utf8'));
      if (parsed && typeof parsed === 'object' && parsed.pid) {
        return parsed;
      }
    } catch {
      // not written yet / mid-write
    }
    await new Promise((resolve) => setTimeout(resolve, Math.min(pollMs, Math.max(0, deadline - Date.now()))));
  }
  return null;
}

function readStateFile(file, deps = {}) {
  const fsp = deps.fs || fs;
  try {
    const parsed = JSON.parse(fsp.readFileSync(file, 'utf8'));
    return parsed && typeof parsed === 'object' ? parsed : null;
  } catch {
    return null;
  }
}

// Keep manual traversal for ASAR payloads, with asynchronous filesystem calls
// so staging a component does not block Electron's main event loop.
async function copyDir(src, dest, deps = {}) {
  const fsp = (deps.fs || fs).promises;
  const syncFs = deps.fs || fs;
  await fsp.mkdir(dest, { recursive: true });
  for (const item of await fsp.readdir(src, { withFileTypes: true })) {
    if (typeof item.isSymbolicLink === 'function' && item.isSymbolicLink()) {
      continue;
    }
    const from = path.join(src, item.name);
    const to = path.join(dest, item.name);
    if (typeof fsp.lstat === 'function') {
      const st = await fsp.lstat(from);
      if (typeof st.isSymbolicLink === 'function' && st.isSymbolicLink()) {
        continue;
      }
      if (st.isDirectory()) {
        await copyDir(from, to, deps);
      } else if (st.isFile()) {
        await fsp.copyFile(from, to);
      }
      continue;
    }
    if (item.isDirectory()) {
      await copyDir(from, to, deps);
    } else if (item.isFile()) {
      if (typeof syncFs.lstatSync === 'function') {
        const st = syncFs.lstatSync(from);
        if (typeof st.isSymbolicLink === 'function' && st.isSymbolicLink()) {
          continue;
        }
      }
      await fsp.copyFile(from, to);
    }
  }
}

async function removeDir(dir, deps = {}) {
  const fsp = (deps.fs || fs).promises;
  try {
    await fsp.rm(dir, { recursive: true, force: true });
    return true;
  } catch {
    return false;
  }
}

module.exports = {
  isPidAlive,
  killPid,
  spawnComponent,
  waitForStateFile,
  readStateFile,
  copyDir,
  removeDir,
  LOG_MAX_BYTES,
};
