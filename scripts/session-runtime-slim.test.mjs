import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { applyReleaseSlim, heavyPackagesToSlim, criticalPackagesToPreserve } from './session-runtime.mjs';
import { emitRuntimeInclude } from './session-runtime-nsis.mjs';
import { inventoryGraph } from './session-runtime-graph.mjs';

function linkDirectory(target, link) {
  fs.mkdirSync(path.dirname(link), { recursive: true });
  fs.symlinkSync(target, link, process.platform === 'win32' ? 'junction' : 'dir');
}

test('applyReleaseSlim accurately identifies and strips 4 heavy packages while preserving critical instances', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'whale-slim-test-'));
  try {
    const stage = path.join(root, 'stage');
    const vendor = path.join(stage, 'runtime/vendor');
    const harness = path.join(vendor, 'deepseek-harness');
    const dshInstances = path.join(harness, 'node_modules/.dsh-instances');

    // Setup instances
    const instLibre = path.join(dshInstances, 'inst-libre-root');
    const instLibreWin = path.join(dshInstances, 'inst-libre-win');
    const instSherpa = path.join(dshInstances, 'inst-sherpa-node');
    const instSherpaWin = path.join(dshInstances, 'inst-sherpa-win');
    const instSharp = path.join(dshInstances, 'inst-sharp-win');
    const instRegular = path.join(dshInstances, 'inst-regular');

    for (const [dir, name] of [
      [instLibre, '@deepseek-ai/libreoffice-kit'],
      [instLibreWin, '@deepseek-ai/libreoffice-kit-win32-x64'],
      [instSherpa, 'sherpa-onnx-node'],
      [instSherpaWin, 'sherpa-onnx-win-x64'],
      [instSharp, '@img/sharp-win32-x64'],
      [instRegular, 'regular-utility'],
    ]) {
      fs.mkdirSync(dir, { recursive: true });
      fs.writeFileSync(path.join(dir, 'package.json'), JSON.stringify({ name, version: '1.0.0' }));
      fs.writeFileSync(path.join(dir, 'index.js'), `// ${name}`);
    }

    // Setup links
    const linkLibre = path.join(harness, 'node_modules/@deepseek-ai/libreoffice-kit');
    const linkLibreWin = path.join(harness, 'node_modules/@deepseek-ai/libreoffice-kit-win32-x64');
    const linkSherpa = path.join(harness, 'node_modules/sherpa-onnx-node');
    const linkSherpaWin = path.join(harness, 'node_modules/sherpa-onnx-win-x64');
    const linkSharp = path.join(harness, 'node_modules/@img/sharp-win32-x64');
    const linkRegular = path.join(harness, 'node_modules/regular-utility');

    linkDirectory(instLibre, linkLibre);
    linkDirectory(instLibreWin, linkLibreWin);
    linkDirectory(instSherpa, linkSherpa);
    linkDirectory(instSherpaWin, linkSherpaWin);
    linkDirectory(instSharp, linkSharp);
    linkDirectory(instRegular, linkRegular);

    const initialLinks = [
      { path: 'node_modules/@deepseek-ai/libreoffice-kit', target: 'node_modules/.dsh-instances/inst-libre-root' },
      { path: 'node_modules/@deepseek-ai/libreoffice-kit-win32-x64', target: 'node_modules/.dsh-instances/inst-libre-win' },
      { path: 'node_modules/sherpa-onnx-node', target: 'node_modules/.dsh-instances/inst-sherpa-node' },
      { path: 'node_modules/sherpa-onnx-win-x64', target: 'node_modules/.dsh-instances/inst-sherpa-win' },
      { path: 'node_modules/@img/sharp-win32-x64', target: 'node_modules/.dsh-instances/inst-sharp-win' },
      { path: 'node_modules/regular-utility', target: 'node_modules/.dsh-instances/inst-regular' },
    ];

    fs.writeFileSync(path.join(harness, '.dsh-runtime-links.json'), JSON.stringify({ version: 1, links: initialLinks }, null, 2));
    fs.writeFileSync(path.join(vendor, 'host-session-runtime.json'), JSON.stringify({ schemaVersion: 1, relocatedLinks: 6 }, null, 2));

    const result = await applyReleaseSlim(vendor);

    // Verify 4 heavy packages removed
    assert.equal(result.removedPackages.length, 4);
    for (const p of heavyPackagesToSlim) {
      assert.ok(result.removedPackages.includes(p));
    }
    assert.equal(result.removedInstances.length, 4);
    assert.equal(result.removedLinks, 4);
    assert.equal(result.remainingLinks, 2);

    // Verify filesystem state
    assert.equal(fs.existsSync(instLibre), false);
    assert.equal(fs.existsSync(instLibreWin), false);
    assert.equal(fs.existsSync(instSherpa), false);
    assert.equal(fs.existsSync(instSherpaWin), false);
    assert.equal(fs.existsSync(instSharp), true);
    assert.equal(fs.existsSync(instRegular), true);

    // Verify links on disk
    assert.equal(fs.existsSync(linkLibre), false);
    assert.equal(fs.existsSync(linkLibreWin), false);
    assert.equal(fs.existsSync(linkSherpa), false);
    assert.equal(fs.existsSync(linkSherpaWin), false);
    assert.equal(fs.existsSync(linkSharp), true);
    assert.equal(fs.existsSync(linkRegular), true);

    // Verify link manifest updated
    const updatedManifest = JSON.parse(fs.readFileSync(path.join(harness, '.dsh-runtime-links.json'), 'utf8'));
    assert.equal(updatedManifest.links.length, 2);
    assert.equal(updatedManifest.links[0].target, 'node_modules/.dsh-instances/inst-sharp-win');
    assert.equal(updatedManifest.links[1].target, 'node_modules/.dsh-instances/inst-regular');

    // Verify host manifest updated
    const updatedHost = JSON.parse(fs.readFileSync(path.join(vendor, 'host-session-runtime.json'), 'utf8'));
    assert.equal(updatedHost.relocatedLinks, 2);

    // Write runtime-graph.json and verify emitRuntimeInclude works cleanly
    const graph = inventoryGraph(vendor, ['runtime-graph.json']);
    const closure = { schemaVersion: 1, files: graph.files, links: graph.links };
    fs.writeFileSync(path.join(vendor, 'runtime-graph.json'), JSON.stringify(closure, null, 2));

    const nshFile = path.join(root, 'slim-runtime.nsh');
    const nsisResult = emitRuntimeInclude(stage, nshFile);
    assert.ok(nsisResult.files > 0);
    assert.equal(nsisResult.links, 2);

    const nshContent = fs.readFileSync(nshFile, 'utf8');
    assert.ok(!nshContent.includes('libreoffice-kit'));
    assert.ok(!nshContent.includes('sherpa-onnx'));
    assert.ok(nshContent.includes('inst-sharp-win'));
    assert.ok(nshContent.includes('inst-regular'));
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('applyReleaseSlim rejects when retained instance depends on removed package', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'whale-slim-dep-'));
  try {
    const stage = path.join(root, 'stage');
    const vendor = path.join(stage, 'runtime/vendor');
    const harness = path.join(vendor, 'deepseek-harness');
    const dshInstances = path.join(harness, 'node_modules/.dsh-instances');

    const instLibre = path.join(dshInstances, 'inst-libre-root');
    const instRegular = path.join(dshInstances, 'inst-regular');

    for (const [dir, name] of [
      [instLibre, '@deepseek-ai/libreoffice-kit'],
      [instRegular, 'regular-utility'],
    ]) {
      fs.mkdirSync(dir, { recursive: true });
      fs.writeFileSync(path.join(dir, 'package.json'), JSON.stringify({ name, version: '1.0.0' }));
    }

    const initialLinks = [
      { path: 'node_modules/@deepseek-ai/libreoffice-kit', target: 'node_modules/.dsh-instances/inst-libre-root' },
      { path: 'node_modules/.dsh-instances/inst-regular/node_modules/dep', target: 'node_modules/.dsh-instances/inst-libre-root' },
    ];

    fs.writeFileSync(path.join(harness, '.dsh-runtime-links.json'), JSON.stringify({ version: 1, links: initialLinks }, null, 2));

    await assert.rejects(
      applyReleaseSlim(vendor),
      /Safety violation: retained instance inst-regular depends on removed target/
    );
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
