const { spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const { loadWorkspaceAuthority, isPathInside } = require('./workspace-authority');

let workspaceAuthority = null;

/** Test seam: pin the trust root (node:test runs outside Electron). */
function setWorkspaceAuthority(authority) {
  workspaceAuthority = authority;
}

/**
 * Authorize a renderer-supplied cwd against the boot workspace and every
 * harness-registered workspace root.
 * @param {unknown} cwd
 * @returns {string | null}
 */
function resolveAuthorizedCwd(cwd) {
  if (workspaceAuthority === null) workspaceAuthority = loadWorkspaceAuthority();
  return workspaceAuthority.resolveAuthorizedCwd(cwd);
}

function asCwd(cwd) {
  return resolveAuthorizedCwd(cwd);
}

/** Resolve a renderer-supplied relative path inside its authorized workspace root. */
function resolveInsideWorkspace(cwd, relativePath) {
  if (workspaceAuthority === null) workspaceAuthority = loadWorkspaceAuthority();
  return workspaceAuthority.resolveInside(cwd, relativePath);
}

/** Wall-clock limit for status/list/read git children. */
const GIT_TIMEOUT_MS = 60_000;
/** Commit timeout: leftover / husky / pre-push may run for minutes. */
const COMMIT_TIMEOUT_MS = 10 * 60_000;
/** Background fetch must not stall the titlebar. */
const FETCH_TIMEOUT_MS = 5_000;
/** Timeout for `gh repo view` / `gh pr list`. */
const GH_TIMEOUT_MS = 30_000;
/** Retained stdout cap; overflow kills the child and sets truncated. */
const GIT_MAX_OUTPUT_BYTES = 2 * 1024 * 1024;
/** Prepared-commit patch cap. */
const PREPARED_COMMIT_PATCH_MAX_OUTPUT_BYTES = 49_000;
/** Range-diff patch cap. */
const RANGE_DIFF_PATCH_MAX_OUTPUT_BYTES = 59_000;
const OUTPUT_TRUNCATED_MARKER = '\n\n[truncated]';

function withTruncationMarker(text, truncated) {
  const raw = String(text || '');
  if (!truncated) return raw;
  return raw.endsWith('[truncated]') ? raw : `${raw}${OUTPUT_TRUNCATED_MARKER}`;
}

/**
 * Env for a git child. Uses the full `process.env` (no
 * `GIT_CEILING_DIRECTORIES`) and strips Electron npm_config_* so leftover
 * / npx do not inherit `electron-skip-binary-download`.
 * @returns {NodeJS.ProcessEnv}
 */
function gitChildEnv() {
  // Do not inherit the desktop's full environment: Git hooks, credential
  // helpers, and remote helpers can read every inherited variable. Keep only
  // the platform/runtime values needed to locate Git and its config; auth is
  // handled by the user's Git credential manager, not ambient tokens.
  const env = {};
  for (const key of [
    'PATH', 'Path', 'PATHEXT', 'SystemRoot', 'ComSpec', 'WINDIR',
    'HOME', 'USERPROFILE', 'HOMEDRIVE', 'HOMEPATH',
    'APPDATA', 'LOCALAPPDATA', 'TEMP', 'TMP', 'TMPDIR',
    'LANG', 'LC_ALL', 'TERM',
  ]) {
    if (typeof process.env[key] === 'string' && process.env[key] !== '') {
      env[key] = process.env[key];
    }
  }
  env.GIT_TERMINAL_PROMPT = '0';
  env.GIT_ASKPASS = '';
  env.GCM_INTERACTIVE = 'never';
  env.SSH_ASKPASS = '';
  env.SSH_ASKPASS_REQUIRE = 'never';
  env.LC_ALL = 'C';
  env.GIT_CONFIG_NOSYSTEM = '1';
  return env;
}

/**
 * Kill a spawned git/gh child and, on Windows, its whole process tree.
 * `child.kill()` on win32 terminates only the direct process; git routinely
 * has live descendants (hooks, ssh, credential helpers) that would otherwise
 * survive a timeout or output-overflow kill and keep holding index locks.
 * `taskkill /T` walks the tree; `/F` is required because a killed git child
 * does not pump messages. POSIX stays on the default SIGTERM.
 * @param {import('node:child_process').ChildProcess} child
 * @param {{ platform?: NodeJS.Platform, spawnKiller?: typeof spawn }} [options] test seams.
 */
function killProcessTree(child, { platform = process.platform, spawnKiller = spawn } = {}) {
  if (platform === 'win32' && typeof child.pid === 'number') {
    try {
      const killer = spawnKiller('taskkill', ['/PID', String(child.pid), '/T', '/F'], {
        windowsHide: true,
        stdio: 'ignore',
      });
      // taskkill missing or unrunnable → at least kill the direct child.
      killer.on('error', () => { child.kill(); });
      // taskkill ran but failed (access denied, and 128 = tree already gone):
      // SIGTERM the direct child so the git process itself never survives a
      // timeout. Killing an already-dead child is a no-op.
      killer.on('exit', (code) => {