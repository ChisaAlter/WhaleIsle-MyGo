'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const store = require('./store');
const registry = require('./registry');
const lifecycle = require('./lifecycle');
const { createComponentsService } = require('./index');

test('rejects version path traversal in store, registry, and service', () => {
  for (const bad of ['', '.', '..', '../evil', '..\\evil', '1.0.0/../../x', '1.0.0\\x', '1.0\0.0', ' 1.0.0']) {
    assert.equal(store.validComponentVersion(bad), false, `expected invalid version: ${JSON.stringify(bad)}`);
    assert.throws(() => store.versionDir('/tmp/comp', 'demo', bad), /invalid component version/);
  }
  assert.equal(store.validComponentVersion('1.2.3-beta.1+build42'), true);

  const normalized = store.normalizeRegistry({
    components: {
      demo: { id: 'demo', version: '../outside', state: 'installed' },
      good: { id: 'good', version: '1.0.0', previous: '../../escape', state: 'installed' },
    },
  });
  assert.equal(normalized.components.demo, undefined);
  assert.equal(normalized.components.good.version, '1.0.0');
  assert.equal(normalized.components.good.previous, null);

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'generic-comp-ver-'));
  try {
    const dir = path.join(tmp, 'demo');
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(path.join(dir, 'index.js'), 'console.log(1);');
    fs.writeFileSync(path.join(dir, 'manifest.json'), JSON.stringify({
      id: 'demo',
      version: '../escape',
      entry: 'index.js',
    }));
    assert.equal(registry.readManifest(dir), null);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
});

test('saveRegistry performs atomic rename with retries and never falls back to direct overwrite', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'generic-atomic-'));
  try {
    const file = store.registryFile(root);
    store.saveRegistry(root, { components: { demo: { id: 'demo', version: '1.0.0', state: 'installed' } } });
    const original = fs.readFileSync(file, 'utf8');

    // Transient EBUSY succeeds on retry.
    let calls = 0;
    const retryFs = {
      ...fs,
      renameSync(from, to) {
        calls += 1;
        if (calls < 3) {
          const err = new Error('busy');
          err.code = 'EBUSY';
          throw err;
        }
        return fs.renameSync(from, to);
      },
    };
    store.saveRegistry(root, { components: { demo: { id: 'demo', version: '1.0.1', state: 'installed' } } }, { fs: retryFs });
    assert.equal(calls, 3);
    assert.match(fs.readFileSync(file, 'utf8'), /1\.0\.1/);

    // Persistent rename failure throws, leaves original intact, and cleans up temp files.
    const beforeFail = fs.readFileSync(file, 'utf8');
    const failingFs = {
      ...fs,
      renameSync() {
        const err = new Error('rename forbidden');
        err.code = 'EPERM';
        throw err;
      },
    };
    assert.throws(() => {
      store.saveRegistry(root, { components: { demo: { id: 'demo', version: '2.0.0', state: 'installed' } } }, { fs: failingFs });
    }, /rename forbidden/);
    assert.equal(fs.readFileSync(file, 'utf8'), beforeFail);
    const leftovers = fs.readdirSync(root).filter((name) => name.includes('.tmp'));
    assert.deepEqual(leftovers, []);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('safeEntryPath rejects symlink escapes via realpathSync.native', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'generic-escape-'));
  try {
    const compDir = path.join(root, 'comp');
    const outsideFile = path.join(root, 'outside.js');
    fs.mkdirSync(compDir, { recursive: true });
    fs.writeFileSync(outsideFile, 'evil();');

    const mockFs = {
      ...fs,
      realpathSync: Object.assign(
        (target) => {
          if (path.resolve(target) === path.resolve(compDir, 'link.js')) {
            return outsideFile;
          }
          return fs.realpathSync(target);
        },
        {
          native(target) {
            if (path.resolve(target) === path.resolve(compDir, 'link.js')) {
              return outsideFile;
            }
            return fs.realpathSync.native(target);
          },
        },
      ),
    };
    assert.equal(registry.safeEntryPath(compDir, 'link.js', { fs: mockFs }), '');
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('spawnComponent strips sensitive installer, node, and token environment variables', () => {
  const savedEnv = {
    DSH_DESKTOP_INSTALL_KEY: process.env.DSH_DESKTOP_INSTALL_KEY,
    NODE_OPTIONS: process.env.NODE_OPTIONS,
    MAGPIE_WEB_KEY: process.env.MAGPIE_WEB_KEY,
    WHALEISLE_CONTROL_TOKEN: process.env.WHALEISLE_CONTROL_TOKEN,
    DSH_AUTH_TOKEN: process.env.DSH_AUTH_TOKEN,
  };
  process.env.DSH_DESKTOP_INSTALL_KEY = 'secret-install';
  process.env.NODE_OPTIONS = '--require=malicious';
  process.env.MAGPIE_WEB_KEY = 'web-secret';
  process.env.WHALEISLE_CONTROL_TOKEN = 'control-secret';
  process.env.DSH_AUTH_TOKEN = 'auth-secret';

  try {
    let capturedOpts = null;
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'generic-env-'));
    try {
      lifecycle.spawnComponent({
        entryFile: path.join(tmp, 'index.js'),
        dir: tmp,
        dataDir: path.join(tmp, 'data'),
        logFile: path.join(tmp, 'data', 'component.log'),
        id: 'demo',
        version: '1.0.0',
      }, {
        fs: {
          mkdirSync() {},
          statSync() { return { size: 0 }; },
          createWriteStream() { return { end() {}, on() {} }; },
        },
        spawn(_cmd, _args, opts) {
          capturedOpts = opts;
          return { pid: 1000, once() {}, stdout: null, stderr: null };
        },
      });
      assert.ok(capturedOpts);
      assert.equal(capturedOpts.env.DSH_DESKTOP_INSTALL_KEY, undefined);
      assert.equal(capturedOpts.env.NODE_OPTIONS, undefined);
      assert.equal(capturedOpts.env.MAGPIE_WEB_KEY, undefined);
      assert.equal(capturedOpts.env.WHALEISLE_CONTROL_TOKEN, undefined);
      assert.equal(capturedOpts.env.DSH_AUTH_TOKEN, undefined);
      assert.equal(capturedOpts.env.ELECTRON_RUN_AS_NODE, '1');
      assert.equal(capturedOpts.env.LAUNCHER_COMPONENT_ID, 'demo');
    } finally {
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  } finally {
    for (const [k, v] of Object.entries(savedEnv)) {
      if (v === undefined) delete process.env[k];
      else process.env[k] = v;
    }
  }
});

test('PID reuse protection refuses to adopt or kill mismatched OS processes', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'generic-pid-'));
  try {
    const samplesRoot = path.join(root, 'samples');
    const compVerDir = path.join(samplesRoot, 'demo', '1.0.0');
    fs.mkdirSync(compVerDir, { recursive: true });
    const entry = path.join(compVerDir, 'index.js');
    fs.writeFileSync(entry, 'setInterval(() => {}, 1000);');
    fs.writeFileSync(path.join(compVerDir, 'manifest.json'), JSON.stringify({
      id: 'demo',
      version: '1.0.0',
      entry: 'index.js',
    }));

    let killedForeign = false;
    const compRoot = path.join(root, 'userData', 'components');
    const activeDir = store.versionDir(compRoot, 'demo', '1.0.0');
    fs.mkdirSync(activeDir, { recursive: true });
    const activeEntry = path.join(activeDir, 'index.js');
    fs.writeFileSync(activeEntry, 'setInterval(() => {}, 1000);');
    fs.writeFileSync(path.join(activeDir, 'manifest.json'), JSON.stringify({
      id: 'demo',
      version: '1.0.0',
      entry: 'index.js',
    }));
    store.saveRegistry(compRoot, {
      components: {
        demo: {
          id: 'demo',
          version: '1.0.0',
          state: 'running',
          pid: 54321,
          url: 'http://127.0.0.1:9999/',
        },
      },
    });

    const foreignDeps = {
      componentsRoot: compRoot,
      samplesRoot,
      platform: 'linux',
      processKill(pid, sig) {
        if (sig === 0 && pid === 54321) return true;
        throw new Error('unexpected processKill');
      },
      inspectProcess(pid) {
        assert.equal(pid, 54321);
        return { executablePath: '/usr/bin/unrelated', commandLine: '/usr/bin/unrelated --daemon' };
      },
      execFileSync() {
        killedForeign = true;
      },
    };

    assert.equal(lifecycle.isPidAlive(54321, { ...foreignDeps, expectedProcess: { id: 'demo', entryFile: activeEntry } }), false);
    assert.equal(lifecycle.killPid(54321, { force: true, expectedProcess: { id: 'demo', entryFile: activeEntry } }, foreignDeps), false);
    assert.equal(killedForeign, false);

    // Reconciliation must mark the reused PID as stopped rather than adopting it.
    const svc = createComponentsService(foreignDeps);
    const listed = svc.list();
    assert.equal(listed.components[0].state, 'stopped');
    assert.equal(listed.components[0].pid, null);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
