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
const MUTATIONS = new Set(['gitCommit', 'gitSwitchBranch', 'gitCreateBranch']);
const GATED_TODO = new Set([
  'gitPush', 'gitPull', 'gitCreateChangeRequest', 'saveConfig', 'openSettings', 'openGallery',
]);
const READ_BODY = Symbol('readBody');

function tokenFromRequest(headers) {
  const authorization = String((headers && (headers.authorization || headers.Authorization)) || '');
  const bearer = authorization.match(/^Bearer\s+(\S+)/i);
  if (bearer) return bearer[1];
  const cookie = String((headers && (headers.cookie || headers.Cookie)) || '');
  for (const part of cookie.split(';')) {
    const index = part.indexOf('=');
    if (index < 0) continue;
    if (part.slice(0, index).trim() === COOKIE_NAME) {
      return decodeURIComponent(part.slice(index + 1).trim());
    }
  }
  return '';
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

function insideRoots(cwd, roots) {
  const path = require('node:path');
  const root = path.resolve(String(cwd || ''));
  return (roots || []).some((entry) => {
    const allowed = path.resolve(String(entry));
    return root === allowed || root.startsWith(allowed + path.sep);
  });
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
  const root = path.resolve(String(cwd || ''));
  const allowed = (roots || []).map((entry) => path.resolve(String(entry)));
  const inside = allowed.some((entry) => root === entry || root.startsWith(entry + path.sep));
  if (!allowed.length || !inside) {
    return { ok: false, error: '工作区不可用' };
  }
  const target = path.resolve(root, String(relativePath || ''));
  if (target !== root && !target.startsWith(root + path.sep)) {
    return { ok: false, error: '工作区不可用' };
  }
  try {
    const entries = fs.readdirSync(target, { withFileTypes: true }).slice(0, 512).map((entry) => ({
      name: entry.name,
      type: entry.isDirectory() ? 'directory' : 'file',
    }));
    return { ok: true, cwd: target, entries };
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
    if (!matchesAny(tokenFromRequest(req.headers || {}), tokens || [])) {
      send(res, 401, 'unauthorized', 'text/plain; charset=utf-8');
      return true;
    }
    if (!READ_ONLY.has(name) && !MUTATIONS.has(name) && !GATED_TODO.has(name)) {
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
        default:
          // TODO(G46): push/pull/PR/saveConfig/open* need desktop services that
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
  GATED_TODO,
  tokenFromRequest,
  tokensEqual,
  shellNameFromUrl,
  createShellHandler,
  wrapResult,
  mountShellRoutes,
  rootedListDir,
};
