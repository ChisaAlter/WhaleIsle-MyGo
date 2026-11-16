'use strict';

// Payload/state layout for the launcher components platform.
//   <userData>/components/registry.json          — installed records
//   <userData>/components/<id>/versions/<v>/     — immutable payload per version
//   <userData>/components/<id>/data/             — per-component data dir,
//                                                  survives update/rollback/uninstall
//
// A dedicated file was chosen over the `components` config field: records
// carry fast-changing runtime state (pid/state/lastError) that would churn
// config.json on every transition, and the 64KiB whitelist cap exists for
// renderer-owned blobs, not for main-process lifecycle bookkeeping.
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

const ID_PATTERN = /^[a-z0-9][a-z0-9._-]{0,63}$/i;
const VERSION_PATTERN = /^[a-z0-9][a-z0-9._+-]{0,127}$/i;
const RECORD_STATES = new Set(['installed', 'stopped', 'running', 'error']);

function validComponentId(id) {
  return typeof id === 'string' && ID_PATTERN.test(id);
}

function validComponentVersion(version) {
  return typeof version === 'string'
    && VERSION_PATTERN.test(version)
    && version !== '.'
    && version !== '..'
    && !version.includes('..')
    && !/[\/\\\x00]/.test(version);
}

function resolveRealSync(fsp, target) {
  if (fsp && fsp.realpathSync) {
    if (typeof fsp.realpathSync.native === 'function') {
      return fsp.realpathSync.native(target);
    }
    return fsp.realpathSync(target);
  }
  return path.resolve(target);
}

function resolveExistingAncestorRealpath(fsp, target) {
  let current = path.resolve(target);
  const trailing = [];
  while (true) {
    try {
      const real = resolveRealSync(fsp, current);
      return trailing.length ? path.join(real, ...trailing.reverse()) : real;
    } catch (error) {
      if (error && error.code !== 'ENOENT' && error.code !== 'ENOTDIR') {
        throw error;
      }
      const parent = path.dirname(current);
      if (parent === current) {
        return path.resolve(target);
      }
      trailing.push(path.basename(current));
      current = parent;
    }
  }
}

function isContainedPath(baseDir, targetPath, { allowEqual = false, fsp = fs } = {}) {
  const realBase = resolveExistingAncestorRealpath(fsp, baseDir);
  const realTarget = resolveExistingAncestorRealpath(fsp, targetPath);
  const rel = path.relative(realBase, realTarget);
  if (rel === '') {
    return allowEqual;
  }
  return !rel.startsWith('..') && !path.isAbsolute(rel);
}

function defaultUserDataDir() {
  try {
    const { app } = require('electron');
    return app.getPath('userData');
  } catch {
    return '';
  }
}

// The launcher's OWN userData — never the managed desktop runtime dir and
// never inside a dsh-home / profiles tree.
function componentsRoot(deps = {}) {
  if (deps.componentsRoot) {
    return deps.componentsRoot;
  }
  const base = deps.userDataDir || defaultUserDataDir();
  return base ? path.join(base, 'components') : '';
}

function registryFile(root) {
  return path.join(root, 'registry.json');
}

function componentDir(root, id) {
  if (!validComponentId(id)) {
    throw new Error(`invalid component id: ${String(id)}`);
  }
  return path.join(root, id);
}

function versionsDir(root, id) {
  return path.join(componentDir(root, id), 'versions');
}

function versionDir(root, id, version) {
  if (!validComponentVersion(version)) {
    throw new Error(`invalid component version: ${String(version)}`);
  }
  return path.join(versionsDir(root, id), version);
}

function stagingDir(root, id) {
  return path.join(versionsDir(root, id), `.staging-${process.pid}-${Date.now()}-${crypto.randomBytes(4).toString('hex')}`);
}

function dataDir(root, id) {
  return path.join(componentDir(root, id), 'data');
}

function stateFile(root, id) {
  return path.join(dataDir(root, id), 'state.json');
}

function logFile(root, id) {
  return path.join(dataDir(root, id), 'component.log');
}

function normalizeRecord(id, raw) {
  if (!raw || typeof raw !== 'object') {
    return null;
  }
  const version = validComponentVersion(raw.version) ? raw.version : '';
  if (!version) {
    return null;
  }
  const previous = validComponentVersion(raw.previous) ? raw.previous : null;
  return {
    id,
    name: typeof raw.name === 'string' ? raw.name.slice(0, 120) : '',
    version,
    previous,
    state: RECORD_STATES.has(raw.state) ? raw.state : 'installed',
    pid: Number.isInteger(raw.pid) && raw.pid > 0 ? raw.pid : null,
    url: typeof raw.url === 'string' ? raw.url.slice(0, 200) : '',
    source: typeof raw.source === 'string' ? raw.source.slice(0, 40) : '',
    installedAt: typeof raw.installedAt === 'string' ? raw.installedAt.slice(0, 40) : '',
    updatedAt: typeof raw.updatedAt === 'string' ? raw.updatedAt.slice(0, 40) : '',
    lastError: typeof raw.lastError === 'string' ? raw.lastError.slice(0, 500) : '',
  };
}

function normalizeRegistry(raw) {
  const out = { components: {} };
  if (!raw || typeof raw !== 'object' || typeof raw.components !== 'object' || !raw.components) {
    return out;
  }
  for (const [id, record] of Object.entries(raw.components)) {
    if (!validComponentId(id)) {
      continue;
    }
    const normalized = normalizeRecord(id, record);
    if (normalized) {
      out.components[id] = normalized;
    }
  }
  return out;
}

// Tolerant read: a torn/corrupt file is moved aside for evidence and the
// platform starts from an empty registry rather than crashing the panel.
function loadRegistry(root, deps = {}) {
  const readFile = deps.readFileSync || fs.readFileSync.bind(fs);
  const rename = deps.renameSync || fs.renameSync.bind(fs);
  const file = registryFile(root);
  let raw;
  try {
    raw = readFile(file, 'utf8');
  } catch {
    return { components: {}, corrupt: false };
  }
  try {
    return { ...normalizeRegistry(JSON.parse(raw)), corrupt: false };
  } catch {
    try {
      rename(file, `${file}.broken`);
    } catch {
      // evidence copy best effort
    }
    return { components: {}, corrupt: true };
  }
}

function sleepSyncMs(ms) {
  try {
    Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms);
  } catch {
    // ignore if SharedArrayBuffer unavailable
  }
}

// Atomic write: unique same-directory temp file + fsync + bounded rename retry
// so a crash or Windows transient lock never corrupts registry.json.
function saveRegistry(root, registry, deps = {}) {
  const fsp = deps.fs || fs;
  const file = registryFile(root);
  if (typeof fsp.lstatSync === 'function') {
    try {
      const existing = fsp.lstatSync(file);
      if (existing && typeof existing.isSymbolicLink === 'function' && existing.isSymbolicLink()) {
        throw new Error('refusing to overwrite symlinked component registry');
      }
    } catch (error) {
      if (error && error.code !== 'ENOENT') {
        throw error;
      }
    }
  }
  const tmp = `${file}.${process.pid}.${crypto.randomBytes(8).toString('hex')}.tmp`;
  const payload = `${JSON.stringify({ components: registry.components || {} }, null, 2)}\n`;
  fsp.mkdirSync(root, { recursive: true });
  let committed = false;
  try {
    if (typeof fsp.openSync === 'function' && typeof fsp.writeSync === 'function' && typeof fsp.closeSync === 'function') {
      const fd = fsp.openSync(tmp, 'wx', 0o600);
      try {
        fsp.writeSync(fd, payload, 0, 'utf8');
        if (typeof fsp.fsyncSync === 'function') {
          fsp.fsyncSync(fd);
        }
      } finally {
        fsp.closeSync(fd);
      }
    } else {
      fsp.writeFileSync(tmp, payload, { mode: 0o600 });
    }
    let lastError = null;
    for (let attempt = 0; attempt < 6; attempt += 1) {
      try {
        fsp.renameSync(tmp, file);
        committed = true;
        return;
      } catch (error) {
        lastError = error;
        if (!error || (error.code !== 'EPERM' && error.code !== 'EBUSY' && error.code !== 'EACCES') || attempt === 5) {
          throw error;
        }
        sleepSyncMs(15 * (attempt + 1));
      }
    }
    if (lastError) {
      throw lastError;
    }
  } finally {
    if (!committed) {
      try {
        if (typeof fsp.rmSync === 'function') {
          fsp.rmSync(tmp, { force: true });
        } else if (typeof fsp.unlinkSync === 'function') {
          fsp.unlinkSync(tmp);
        }
      } catch {
        // tmp cleanup best effort
      }
    }
  }
}

module.exports = {
  validComponentId,
  validComponentVersion,
  resolveExistingAncestorRealpath,
  isContainedPath,
  componentsRoot,
  registryFile,
  componentDir,
  versionsDir,
  versionDir,
  stagingDir,
  dataDir,
  stateFile,
  logFile,
  normalizeRegistry,
  loadRegistry,
  saveRegistry,
};
