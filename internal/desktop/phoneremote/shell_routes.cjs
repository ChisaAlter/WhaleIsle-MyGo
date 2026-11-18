'use strict';
/**
 * Desktop shell routes for the phone/remote UI: POST /__remote__/shell/<name>.
 *
 * Native authority: reference/electron/src/main/remote.js:708-730 (auth before
 * dispatch, POST only), 779-807 (404 unknown name, 501 when no invokeShell, 413
 * body >256 KiB, 400 invalid JSON, else {ok,result}); the route table lives in
 * remote-shell.js:1-109 and the token rules in shared/remote-auth.js:45-64
 * (Authorization: Bearer <token> or cookie dsh_remote=<token>).
 *
 * MyGo mounts the subset it can serve safely on the daemon's mobile web server:
 *   read-only  gitStatus, gitFetchForStatus, gitDiff, gitBranchList, listDir,
 *              getConfig  -> served through the existing git gate / a rooted fs
 *   mutations  gitCommit, gitPush, gitPull, gitSwitchBranch, gitCreateBranch,
 *              gitCreateChangeRequest, saveConfig, open* -> native "unavailable"
 *              shape (501) with a TODO; never bypasses authorisation
 */

const COOKIE_NAME = 'dsh_remote';
const MAX_BODY = 262144;
const READ_ONLY = new Set(['gitStatus', 'gitFetchForStatus', 'gitDiff', 'gitBranchList', 'listDir', 'getConfig']);
const MUTATIONS = new Set(['gitCommit', 'gitSwitchBranch', 'gitCreateBranch', 'gitPush', 'gitPull', 'gitCreateChangeRequest']);
const HOST_ACTIONS = new Set(['openSettings', 'openGallery', 'saveConfig']);
// Every native shell name is now served; keep the set for forward-compatible names.
const GATED_TODO = new Set();
const READ_BODY = Symbol('readBody');

function headerValue(headers, name) {
  if (!headers || typeof headers !== 'object') return '';
  const target = name.toLowerCase();
  for (const key of Object.keys(headers)) {
    if (key.toLowerCase() === target) {
      const value = headers[key];
      return Array.isArray(value) ? String(value[0] || '') : String(value || '');
    }
  }
  return '';
}

function authFromRequest(headers) {
  const authorization = headerValue(headers, 'authorization');
  const bearer = authorization.match(/^Bearer\s+(\S+)/i);
  if (bearer) return { token: bearer[1], viaCookie: false };
  const cookie = headerValue(headers, 'cookie');
  for (const part of cookie.split(';')) {
    const index = part.indexOf('=');
    if (index < 0) continue;
    if (part.slice(0, index).trim() === COOKIE_NAME) {
      try {
        return { token: decodeURIComponent(part.slice(index + 1).trim()), viaCookie: true };
      } catch {
        return { token: '', viaCookie: true };
      }
    }
  }
  return { token: '', viaCookie: false };
}

function tokenFromRequest(headers) {
  return authFromRequest(headers).token;
}

function normalizedUrlHost(url) {
  const port = url.port || (url.protocol === 'https:' ? '443' : url.protocol === 'http:' ? '80' : '');
  return `${url.hostname.toLowerCase()}:${port}`;
}

function cookieCsrfOk(headers) {
  const originRaw = headerValue(headers, 'origin').trim();
  if (originRaw) {
    if (originRaw === 'null') return false;
    let originUrl;
    try {
      originUrl = new URL(originRaw);
    } catch {
      return false;
    }
    if (originUrl.protocol !== 'http:' && originUrl.protocol !== 'https:') return false;
    if (originUrl.username || originUrl.password || (originUrl.pathname !== '/' && originUrl.pathname !== '') || originUrl.search || originUrl.hash) {
      return false;
    }
    const hostRaw = headerValue(headers, 'host').trim();
    if (!hostRaw || /[\\/@?#\s]/.test(hostRaw)) return false;
    let hostUrl;
    try {
      hostUrl = new URL(`${originUrl.protocol}//${hostRaw}`);
    } catch {
      return false;
    }
    if (hostUrl.username || hostUrl.password || (hostUrl.pathname !== '/' && hostUrl.pathname !== '') || hostUrl.search || hostUrl.hash || !hostUrl.hostname) {
      return false;
    }
    if (normalizedUrlHost(originUrl) !== normalizedUrlHost(hostUrl)) return false;
  }

  const fetchSiteRaw = headerValue(headers, 'sec-fetch-site').trim();
  if (fetchSiteRaw) {
    const fetchSite = fetchSiteRaw.toLowerCase();
    if (fetchSite !== 'same-origin' && fetchSite !== 'none') return false;
  }

  const contentTypeRaw = headerValue(headers, 'content-type').trim();
  if (contentTypeRaw) {
    const mediaType = contentTypeRaw.split(';', 1)[0].trim().toLowerCase();
    if (mediaType !== 'application/json') return false;
  }

  return true;
}

function tokensEqual(presented, expected) {
  const crypto = require('node:crypto');
  const a = crypto.createHash('sha256').update(String(presented || '')).digest();
  const b = crypto.createHash('sha256').update(String(expected || '')).digest();
  const equal = crypto.timingSafeEqual(a, b);
  // An empty presented token never matches, even when both sides hash equal.
  return Boolean(String(presented || '')) && Boolean(String(expected || '')) && equal;
}

function matchesAny(presented, expected) {
  const list = Array.isArray(expected) ? expected : [expected];
  let match = false;
  for (const candidate of list) {
    // Loop without early exit so the timing does not reveal which token matched.
    match = tokensEqual(presented, candidate) || match;
  }
  return match;
}

/** Path name, or null when the request is not a shell route. */
function shellNameFromUrl(url) {
  const pathname = decodeURIComponent(String(url || '').split('?')[0] || '');
  const match = pathname.match(/^\/__remote__\/shell\/([^/]+)$/);
  return match ? match[1] : null;
}

function send(res, status, body, contentType) {
  const payload = typeof body === 'string' ? body : JSON.stringify(body);
  res.writeHead(status, { 'content-type': contentType || 'application/json; charset=utf-8' });
  res.end(payload);
}

function fail(res, status, error) {
  send(res, status, { ok: false, error });
}

// Native remote-shell.js:39-45 wrapResult: null -> empty-message failure, an
// explicit {ok:false} result becomes a failure with its message.
function wrapResult(res, result, emptyMessage) {
  if (result == null) {
    fail(res, 400, emptyMessage || '工作区不可用');
    return;
  }
  if (result && result.ok === false) {
    fail(res, 400, result.message || result.error || emptyMessage || '工作区不可用');
    return;
  }
  send(res, 200, { ok: true, result });
}

function nonEmptyString(value, max) {
  return typeof value === 'string' && value.trim() !== '' && value.length <= (max || 512);
}

function realpathOrNull(fs, targetPath) {
  try {
    const fn = typeof fs.realpathSync.native === 'function' ? fs.realpathSync.native : fs.realpathSync;
    return fn(targetPath);
  } catch {
    return null;
  }
}

function normalizeCompare(path, value) {
  const resolved = path.resolve(String(value || ''));
  return process.platform === 'win32' ? resolved.toLowerCase() : resolved;
}

function isContained(path, rootNorm, targetNorm) {
  if (targetNorm === rootNorm) return true;
  const prefix = rootNorm.endsWith(path.sep) ? rootNorm : rootNorm + path.sep;
  return targetNorm.startsWith(prefix);
}

function hasGitSegment(value) {
  return String(value || '')
    .split(/[\\/]+/)
    .some((part) => part.toLowerCase() === '.git');
}

function findContainingRoot(fs, path, cwd, roots) {
  const cwdText = typeof cwd === 'string' ? cwd.trim() : '';
  if (!cwdText) return null;
  const cwdReal = realpathOrNull(fs, path.resolve(cwdText));
  if (!cwdReal) return null;
  const cwdNorm = normalizeCompare(path, cwdReal);
  for (const entry of roots || []) {
    if (typeof entry !== 'string' || !entry.trim()) continue;
    const rootReal = realpathOrNull(fs, path.resolve(entry));
    if (!rootReal) continue;
    const rootNorm = normalizeCompare(path, rootReal);
    if (!isContained(path, rootNorm, cwdNorm)) continue;
    const relFromRoot = path.relative(rootNorm, cwdNorm);
    if (path.basename(cwdNorm) === '.git' || hasGitSegment(relFromRoot)) continue;
    return { cwdReal, cwdNorm, rootNorm };
  }
  return null;
}

function insideRoots(cwd, roots) {
  const fs = require('node:fs');
  const path = require('node:path');
  return findContainingRoot(fs, path, cwd, roots) !== null;
}

function denial(error) {
  return /not authorized|permission expired|denied|forbidden/i.test(String(error || ''));
}

function readBody(req) {
  return new Promise((resolve, reject) => {
    let size = 0;
    const chunks = [];
    req.on('data', (chunk) => {
      size += chunk.length;
      if (size > MAX_BODY) {
        reject(new Error('too large'));
        req.destroy();
        return;
      }
      chunks.push(chunk);
    });
    req.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')));
    req.on('error', reject);
  });
}

function rootedListDir(cwd, relativePath, roots) {
  const fs = require('node:fs');
  const path = require('node:path');
  const match = findContainingRoot(fs, path, cwd, roots);
  if (!match) {
    return { ok: false, error: '工作区不可用' };
  }
  const relText = relativePath == null ? '' : String(relativePath);
  if (hasGitSegment(relText)) {
    return { ok: false, error: '工作区不可用' };
  }
  const targetLexical = path.resolve(match.cwdReal, relText);
  const targetLexNorm = normalizeCompare(path, targetLexical);
  if (!isContained(path, match.cwdNorm, targetLexNorm)) {
    return { ok: false, error: '工作区不可用' };
  }
  const targetReal = realpathOrNull(fs, targetLexical);
  if (!targetReal) {
    return { ok: false, error: '工作区不可用' };
  }
  const targetNorm = normalizeCompare(path, targetReal);
  const relFromRoot = path.relative(match.rootNorm, targetNorm);
  if (!isContained(path, match.cwdNorm, targetNorm) || path.basename(targetNorm) === '.git' || hasGitSegment(relFromRoot)) {
    return { ok: false, error: '工作区不可用' };
  }
  try {
    const names = fs.readdirSync(targetReal);
    const entries = [];
    for (const name of names) {
      if (entries.length >= 512) break;
      if (String(name).toLowerCase() === '.git') continue;
      const entryPath = path.join(targetReal, name);
      let lstat;
      try {
        lstat = fs.lstatSync(entryPath);
      } catch {
        continue;
      }
      let isDir = lstat.isDirectory();
      if (lstat.isSymbolicLink()) {
        const entryReal = realpathOrNull(fs, entryPath);
        if (!entryReal) continue;
        const entryNorm = normalizeCompare(path, entryReal);
        if (!isContained(path, match.cwdNorm, entryNorm)) continue;
        const entryRelFromRoot = path.relative(match.rootNorm, entryNorm);
        if (path.basename(entryNorm) === '.git' || hasGitSegment(entryRelFromRoot)) continue;
        try {
          isDir = fs.statSync(entryReal).isDirectory();
        } catch {
          continue;
        }
      }
      entries.push({ name, type: isDir ? 'directory' : 'file' });
    }
    return { ok: true, cwd: targetReal, entries };
  } catch (error) {
    return { ok: false, error: String(error && error.message ? error.message : error) };
  }
}

/**
 * Builds the request handler.
 * opts: { tokens(): string[], git, roots(): string[], config(): object }
 * Returns async (req, res) => boolean (true when the request was handled).
 */
function createShellHandler(opts) {
  const options = opts || {};
  return async function handleShellRequest(req, res) {
    const name = shellNameFromUrl(req.url || '');
    if (!name) return false;
    if ((req.method || '') !== 'POST') return false;
    const tokens = typeof options.tokens === 'function' ? options.tokens() : options.tokens;
    const auth = authFromRequest(req.headers || {});
    if (!matchesAny(auth.token, tokens || [])) {
      send(res, 401, 'unauthorized', 'text/plain; charset=utf-8');
      return true;
    }
    if (auth.viaCookie && !cookieCsrfOk(req.headers || {})) {
      fail(res, 403, 'forbidden');
      return true;
    }
    if (!READ_ONLY.has(name) && !MUTATIONS.has(name) && !HOST_ACTIONS.has(name) && !GATED_TODO.has(name)) {
      fail(res, 404, 'not found');
      return true;
    }
    let raw = '';
    try {
      raw = await readBody(req);
    } catch {
      fail(res, 413, '请求过大');
      return true;
    }
    let payload = {};
    if (raw) {
      try {
        payload = JSON.parse(raw);
      } catch {
        fail(res, 400, 'invalid json');
        return true;
      }
    }
    const git = options.git || {};
    try {
      switch (name) {
        case 'gitStatus':
          wrapResult(res, await git.gitStatus(payload.cwd));
          return true;
        case 'gitFetchForStatus':
          wrapResult(res, await git.gitFetchForStatus(payload.cwd));
          return true;
        case 'gitDiff':
          wrapResult(res, await git.gitDiff(payload.cwd, payload.options || {}));
          return true;
        case 'gitBranchList':
          wrapResult(res, await git.gitBranchList(payload.cwd));
          return true;
        case 'listDir': {
          const roots = typeof options.roots === 'function' ? options.roots() : options.roots;
          const result = rootedListDir(payload.cwd, payload.relativePath || '', roots || []);
          send(res, 200, result.ok ? { ok: true, result } : { ok: false, error: result.error });
          return true;
        }
        case 'getConfig': {
          // Tokens and credentials never leave the daemon.
          const config = typeof options.config === 'function' ? options.config() : options.config || {};
          const safe = {
            remoteEnabled: Boolean(config.remoteEnabled),
            remoteMode: String(config.remoteMode || ''),
            remotePort: Number(config.remotePort) || 0,
            remoteBindAddress: String(config.remoteBindAddress || ''),
            remoteLanTls: Boolean(config.remoteLanTls),
            remoteGitEnabled: Array.isArray(config.remoteGitRoots) ? config.remoteGitRoots.length > 0 : false,
          };
          send(res, 200, { ok: true, result: safe });
          return true;
        }
        case 'gitCommit': {
          const roots = typeof options.roots === 'function' ? options.roots() : options.roots;
          if (!nonEmptyString(payload.cwd) || !insideRoots(payload.cwd, roots || [])) {
            fail(res, 403, 'Mobile Git is not authorized for this workspace');
            return true;
          }
          if (!nonEmptyString(payload.message, 65536)) {
            fail(res, 400, 'invalid args: message');
            return true;
          }
          const filePaths = payload.filePaths;
          if (filePaths !== undefined && (!Array.isArray(filePaths) || filePaths.some((f) => !nonEmptyString(f, 4096)))) {
            fail(res, 400, 'invalid args: filePaths');
            return true;
          }
          wrapResult(res, await git.gitCommit(payload.cwd, payload.message, filePaths, undefined, payload.options || {}));
          return true;
        }
        case 'gitSwitchBranch': {
          const roots = typeof options.roots === 'function' ? options.roots() : options.roots;
          if (!nonEmptyString(payload.cwd) || !insideRoots(payload.cwd, roots || [])) {
            fail(res, 403, 'Mobile Git is not authorized for this workspace');
            return true;
          }
          if (!nonEmptyString(payload.ref, 512)) {
            fail(res, 400, 'invalid args: ref');
            return true;
          }
          wrapResult(res, await git.gitSwitchBranch(payload.cwd, payload.ref));
          return true;
        }
        case 'gitCreateBranch': {
          const roots = typeof options.roots === 'function' ? options.roots() : options.roots;
          if (!nonEmptyString(payload.cwd) || !insideRoots(payload.cwd, roots || [])) {
            fail(res, 403, 'Mobile Git is not authorized for this workspace');
            return true;
          }
          if (!nonEmptyString(payload.name, 512)) {
            fail(res, 400, 'invalid args: name');
            return true;
          }
          wrapResult(res, await git.gitCreateBranch(payload.cwd, payload.name));
          return true;
        }
        case 'gitPush':
        case 'gitPull': {
          const roots = typeof options.roots === 'function' ? options.roots() : options.roots;
          if (!nonEmptyString(payload.cwd) || !insideRoots(payload.cwd, roots || [])) {
            fail(res, 403, 'Mobile Git is not authorized for this workspace');
            return true;
          }
          // Native remote-shell.js:70-73 passes only cwd; the branch is HEAD's.
          // Remote/branch are accepted for forward compatibility and validated
          // when present, but the gated helper decides the push/pull target.
          for (const key of ['remote', 'branch']) {
            if (payload[key] !== undefined && !nonEmptyString(payload[key], 512)) {
              fail(res, 400, 'invalid args: ' + key);
              return true;
            }
          }
          wrapResult(res, name === 'gitPush'
            ? await git.gitPush(payload.cwd)
            : await git.gitPull(payload.cwd));
          return true;
        }
        case 'gitCreateChangeRequest': {
          const roots = typeof options.roots === 'function' ? options.roots() : options.roots;
          if (!nonEmptyString(payload.cwd) || !insideRoots(payload.cwd, roots || [])) {
            fail(res, 403, 'Mobile Git is not authorized for this workspace');
            return true;
          }
          const input = payload.input === undefined ? {} : payload.input;
          if (typeof input !== 'object' || input === null || Array.isArray(input)) {
            fail(res, 400, 'invalid args: input');
            return true;
          }
          for (const key of ['title', 'body', 'base', 'branch']) {
            if (input[key] !== undefined && !nonEmptyString(input[key], key === 'body' ? 65536 : 4096)) {
              fail(res, 400, 'invalid args: input.' + key);
              return true;
            }
          }
          // Native git.js:949-969 needs the branch upstream first; a no-op push
          // is harmless and a failure is reported by the create call itself.
          try {
            await git.gitPush(payload.cwd);
          } catch {
            /* the create call owns the final reason */
          }
          wrapResult(res, await git.gitCreateChangeRequest(payload.cwd, input));
          return true;
        }
        case 'openSettings': {
          const section = payload.sectionId === undefined ? '' : payload.sectionId;
          if (section !== '' && !nonEmptyString(section, 200)) {
            fail(res, 400, 'invalid args: sectionId');
            return true;
          }
          // Native remote-shell.js:84-87 returns the desktop's opened flag.
          const host = options.host || {};
          const opened = typeof host.openSettings === 'function'
            ? Boolean(await host.openSettings(section))
            : false;
          send(res, 200, { ok: true, result: { opened, section } });
          return true;
        }
        case 'openGallery': {
          // Native remote-shell.js:88-91 opens the appearance section.
          const host = options.host || {};
          const opened = typeof host.openSettings === 'function'
            ? Boolean(await host.openSettings('appearance'))
            : false;
          send(res, 200, { ok: true, result: { opened, section: 'appearance' } });
          return true;
        }
        case 'saveConfig': {
          const patch = payload.patch === undefined ? {} : payload.patch;
          if (typeof patch !== 'object' || patch === null || Array.isArray(patch)) {
            fail(res, 400, 'invalid args: patch');
            return true;
          }
          const allowed = ['remoteEnabled', 'remoteMode', 'remotePort', 'remoteBindAddress', 'remoteLanTls', 'remoteAppBaseUrl', 'locale', 'theme'];
          for (const key of Object.keys(patch)) {
            if (!allowed.includes(key)) {
              fail(res, 400, 'invalid args: patch.' + key);
              return true;
            }
          }
          const host = options.host || {};
          if (typeof host.saveConfig !== 'function') {
            fail(res, 400, 'saveConfig requires the desktop host');
            return true;
          }
          try {
            const saved = await host.saveConfig(patch);
            if (!saved || saved.ok === false) {
              fail(res, 400, (saved && (saved.error || saved.message)) || 'saveConfig failed');
              return true;
            }
            send(res, 200, { ok: true, result: saved.config || {} });
            return true;
          } catch (error) {
            fail(res, 400, String(error && error.message ? error.message : error));
            return true;
          }
        }
        default:
          // No native shell name reaches this branch; kept as a guard.
          fail(res, 404, 'not found');
          return true;
          // are not proven end to end yet; the native "unavailable" shape keeps
          // the phone honest instead of half-wiring them.
          fail(res, 501, 'shell unavailable');
          return true;
      }
    } catch (error) {
      const message = String(error && error.message ? error.message : error);
      fail(res, denial(message) ? 403 : 400, message);
      return true;
    }
  };
}

/** Prepends the shell handler in front of the server's existing request listeners. */
function mountShellRoutes(server, opts) {
  if (!server || server.__mygoShellMounted) return false;
  const handler = createShellHandler(opts);
  const existing = server.listeners('request');
  server.removeAllListeners('request');
  server.on('request', (req, res) => {
    Promise.resolve(handler(req, res))
      .then((handled) => {
        if (handled) return;
        for (const listener of existing) listener.call(server, req, res);
      })
      .catch(() => {
        if (!res.headersSent) send(res, 500, 'shell failed', 'text/plain; charset=utf-8');
      });
  });
  server.__mygoShellMounted = true;
  return true;
}

module.exports = {
  COOKIE_NAME,
  MAX_BODY,
  READ_ONLY,
  MUTATIONS,
  HOST_ACTIONS,
  GATED_TODO,
  tokenFromRequest,
  tokensEqual,
  shellNameFromUrl,
  createShellHandler,
  wrapResult,
  mountShellRoutes,
  rootedListDir,
};
