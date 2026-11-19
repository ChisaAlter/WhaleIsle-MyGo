import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { applyReleaseSlim, assertCapabilities, capabilityPairs, criticalPackagesToPreserve, optionalNativeEnginesToSlim } from './session-runtime.mjs';
import { emitRuntimeInclude } from './session-runtime-nsis.mjs';
import { inventoryGraph } from './session-runtime-graph.mjs';

function linkDirectory(target, link) {
  fs.mkdirSync(path.dirname(link), { recursive: true });
  fs.symlinkSync(target, link, process.platform === 'win32' ? 'junction' : 'dir');
}

function capabilityNames() {
  const office = capabilityPairs.find(pair => pair.capability === 'office-to-pdf');
  const speech = capabilityPairs.find(pair => pair.capability === 'speech-to-text-sensevoice');
  return {
    officeLoader: office.loader,
    officeWasm: office.engines.find(engine => !engine.optionalNative).name,
    officeNative: office.engines.find(engine => engine.optionalNative).name,
    speechLoader: speech.loader,
    speechEngine: speech.engines[0].name,
  };
}

// One instance store per package, linked the way a staged Host links them.
function stageVendor(root, names, { optional = {}, addons = [], assets = {}, extraLinks = [] } = {}) {
  const stage = path.join(root, 'stage');
  const vendor = path.join(stage, 'runtime/vendor');
  const harness = path.join(vendor, 'deepseek-harness');
  const instances = path.join(harness, 'node_modules/.dsh-instances');
  const links = [];
  const ids = new Map();
  names.forEach((name, position) => {
    const id = `inst-${position + 1}`;
    ids.set(name, id);
    const dir = path.join(instances, id);
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(path.join(dir, 'package.json'), JSON.stringify({ name, optionalDependencies: optional[name] }));
    if (addons.includes(name)) fs.writeFileSync(path.join(dir, 'sherpa-onnx.node'), '// native addon');
    if (assets[name]) {
      fs.mkdirSync(path.dirname(path.join(dir, assets[name])), { recursive: true });
      fs.writeFileSync(path.join(dir, assets[name]), '// engine asset');
    }
    linkDirectory(dir, path.join(harness, 'node_modules', name));
    links.push({ path: `node_modules/${name}`, target: `node_modules/.dsh-instances/${id}` });
  });
  for (const link of extraLinks) links.push(link);
  fs.writeFileSync(path.join(harness, '.dsh-runtime-links.json'), JSON.stringify({ version: 1, links }, null, 2));
  fs.writeFileSync(path.join(vendor, 'host-session-runtime.json'), JSON.stringify({ schemaVersion: 1, relocatedLinks: links.length }, null, 2));
  return { stage, vendor, harness, instances, ids };
}

function instanceIdOf(instances, position) {
  return fs.readdirSync(instances).sort()[position];
}

test('slim keeps every capability loader with a working engine and the installer carries both halves', async () => {
  const names = capabilityNames();
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'whale-slim-capability-'));
  try {
    const { stage, vendor, harness, ids } = stageVendor(root, [
      names.officeLoader, names.officeWasm, names.officeNative,
      names.speechLoader, names.speechEngine,
      '@img/sharp-win32-x64', 'regular-utility',
    ], {
      optional: { [names.officeLoader]: { [names.officeNative]: '*' } },
      addons: [names.speechEngine],
      assets: { [names.officeWasm]: 'assets/dsoffice.wasm' },
    });

    const result = await applyReleaseSlim(vendor);

    assert.deepEqual(optionalNativeEnginesToSlim, [names.officeNative]);
    assert.deepEqual(result.removedPackages, [names.officeNative]);
    for (const kept of [names.officeLoader, names.officeWasm, names.speechLoader, names.speechEngine]) {
      assert.ok(fs.existsSync(path.join(harness, 'node_modules', kept)), `slim removed required package ${kept}`);
    }
    assert.equal(fs.existsSync(path.join(harness, 'node_modules', names.officeNative)), false);

    const report = await assertCapabilities(vendor);
    assert.ok(report.capabilities.every(capability => capability.complete), JSON.stringify(report, null, 2));

    const graph = inventoryGraph(vendor, ['runtime-graph.json']);
    fs.writeFileSync(path.join(vendor, 'runtime-graph.json'), JSON.stringify({ schemaVersion: 1, files: graph.files, links: graph.links }, null, 2));
    const nshFile = path.join(root, 'capability-runtime.nsh');
    emitRuntimeInclude(stage, nshFile);
    const nsh = fs.readFileSync(nshFile, 'utf8');
    for (const kept of [names.officeLoader, names.officeWasm, names.speechLoader, names.speechEngine]) {
      assert.ok(nsh.includes(ids.get(kept)), `installer must carry the instance carrying ${kept}`);
    }
    assert.equal(nsh.includes(ids.get(names.officeNative)), false, 'installer must not carry the slimmed native engine');

    // A loader whose engine never made it into the stage is what this gate exists for.
    const broken = stageVendor(path.join(root, 'broken'), [names.officeLoader, names.officeWasm, names.speechLoader], {
      assets: { [names.officeWasm]: 'assets/dsoffice.wasm' },
    });
    await assert.rejects(
      assertCapabilities(broken.vendor),
      error => error.message.includes('speech-to-text-sensevoice') && error.message.includes(`${names.speechEngine} missing`),
    );
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('slim still refuses a retained instance that hard-depends on a removed engine', async () => {
  const names = capabilityNames();
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'whale-slim-required-dep-'));
  try {
    const { vendor, harness, instances } = stageVendor(root, [names.officeNative, 'regular-utility']);
    const nativeId = instanceIdOf(instances, 0);
    const regularId = instanceIdOf(instances, 1);
    const manifestPath = path.join(harness, '.dsh-runtime-links.json');
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    manifest.links.push({ path: `node_modules/.dsh-instances/${regularId}/node_modules/${names.officeNative}`, target: `node_modules/.dsh-instances/${nativeId}` });
    fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));

    await assert.rejects(applyReleaseSlim(vendor), /retained instance .* depends on removed target/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('slim prunes an optional native engine a retained loader may fall back from', async () => {
  const names = capabilityNames();
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'whale-slim-optional-dep-'));
  try {
    const { vendor, harness, instances } = stageVendor(root, [names.officeLoader, names.officeNative], {
      optional: { [names.officeLoader]: { [names.officeNative]: '*' } },
    });
    const loaderId = instanceIdOf(instances, 0);
    const nativeId = instanceIdOf(instances, 1);
    const manifestPath = path.join(harness, '.dsh-runtime-links.json');
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    manifest.links.push({ path: `node_modules/.dsh-instances/${loaderId}/node_modules/${names.officeNative}`, target: `node_modules/.dsh-instances/${nativeId}` });
    fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));

    const result = await applyReleaseSlim(vendor);
    assert.deepEqual(result.removedPackages, [names.officeNative]);
    assert.equal(fs.existsSync(path.join(harness, 'node_modules', names.officeLoader)), true);
    const updated = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    assert.ok(updated.links.every(link => !link.target.includes(nativeId)), JSON.stringify(updated.links));
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('no capability package may be listed for slimming', () => {
  const preserved = new Set(criticalPackagesToPreserve);
  for (const pair of capabilityPairs) {
    assert.ok(preserved.has(pair.loader), `${pair.loader} must be preserved`);
    for (const engine of pair.engines.filter(candidate => !candidate.optionalNative)) {
      assert.ok(preserved.has(engine.name), `${engine.name} must be preserved`);
    }
  }
  for (const slimmed of optionalNativeEnginesToSlim) {
    assert.equal(preserved.has(slimmed), false, `${slimmed} is both slimmed and preserved`);
  }
});
