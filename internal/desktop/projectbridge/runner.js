'use strict';
// MyGo adapter: only embedded, pinned modules are executable. No caller scripts.
const fs = require('node:fs');
const path = require('node:path');
const Module = require('node:module');
const input = JSON.parse(fs.readFileSync(0, 'utf8'));
const cache = new Map();
function load(name) {
  if (cache.has(name)) return cache.get(name).exports;
  if (!Object.hasOwn(input.sources, name)) throw new Error('Unsupported desktop dependency');
  const mod = new Module(name); cache.set(name, mod);
  mod.filename = name;
  mod.require = target => {
    if (target.startsWith('node:') || ['path', 'fs'].includes(target)) return require(target);
    if (name === 'main/dsh-project-desktop.js' && target === './plugins') return { webProfileDir: () => input.profileDir };
    if (!target.startsWith('.')) throw new Error('Unsupported desktop dependency');
    let resolved = path.posix.normalize(path.posix.join(path.posix.dirname(name), target));
    if (!resolved.endsWith('.js')) resolved += '.js';
    return load(resolved);
  };
  mod._compile(input.sources[name], name); return mod.exports;
}
async function main() {
  if (Number(process.versions.node.split('.')[0]) < 22) throw new Error('Project requires Node.js 22 or newer');
  load('shared/dsh-home.js').setDesktopDshHome(input.home);
  const { createWorkspaceAuthority, filterRegisteredWorkspaceRoots, readHarnessRegisteredWorkspacePaths } = load('main/workspace-authority.js');
  const authority = createWorkspaceAuthority({ workspace: filterRegisteredWorkspaceRoots([input.workspace])[0] || '',
    listRegisteredWorkspaces: () => filterRegisteredWorkspaceRoots(readHarnessRegisteredWorkspacePaths(input.home)) });
  if (input.action === 'ready') {
    load('main/project-environment.js'); load('main/dsh-project-desktop.js');
    return { ok: true, available: true };
  }
  if (input.action === 'overlay') return load('main/dsh-project-desktop.js').ensureDesktopDshProject({ sourceDir: input.sourceDir, profileDir: input.profileDir, harnessRoot: input.harnessRoot });
  const payload = input.payload;
  // Host's create endpoint is authenticated but is not a native directory picker.
  // A new directory must first be chosen/registered through MyGo's explicit UI.
  if (payload.action === 'select-directory' && !authority.resolveAuthorizedCwd(payload.workingDirectory)) {
    throw new Error('请先通过桌面目录选择入口登记此工作目录');
  }
  const { createProjectEnvironment } = load('main/project-environment.js');
  const env = createProjectEnvironment({ home: input.home, authority,
    openPath: async () => { throw new Error('Unsupported: use the native resolved-path opener'); } });
  return env.dispatch(payload);
}
main().then(value => process.stdout.write(JSON.stringify(value))).catch(error => {
  // Never return arbitrary subprocess stderr, credentials, or user filesystem paths.
  const redact = load('main/project-environment.js').redact;
  process.stdout.write(JSON.stringify({ ok: false, error: redact(error.message).slice(0, 2048) }));
});
