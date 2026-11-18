import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

import {stagePlugin, stageThirdPartyPlugins, verifyPluginSource} from './stage-third-party-plugins.mjs';

function fixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'stage-plugin-'));
  const source = path.join(root, 'third_party', 'dsh-deepseek-tools');
  fs.mkdirSync(path.join(source, 'lib'), {recursive: true});
  fs.writeFileSync(path.join(source, 'package.json'), JSON.stringify({
    name: 'dsh-deepseek-tools', main: 'lib/index.js',
    exports: {'.': './lib/index.js', './client': './client/client.js'},
  }));
  fs.writeFileSync(path.join(source, 'lib', 'index.js'), 'export const name = "dsh-deepseek-tools";\n');
  fs.mkdirSync(path.join(source, 'client'), {recursive: true});
  fs.writeFileSync(path.join(source, 'client', 'client.js'), 'export {};\n');
  const vendorDir = path.join(root, 'runtime', 'vendor');
  fs.mkdirSync(vendorDir, {recursive: true});
  return {root, source, vendorDir};
}

test('a stage carries the tracked package under its upstream name', () => {
  const {root, vendorDir} = fixture();
  const staged = stageThirdPartyPlugins({projectRoot: root, vendorDir});
  assert.deepEqual(staged.map(entry => [entry.name, entry.action]), [['dsh-deepseek-tools', 'copied']]);
  assert.ok(fs.statSync(path.join(vendorDir, 'dsh-deepseek-tools', 'lib', 'index.js')).isFile());
});

test('staging twice is a no-op, and a changed source replaces the copy as a unit', () => {
  const {root, source, vendorDir} = fixture();
  stageThirdPartyPlugins({projectRoot: root, vendorDir});
  assert.equal(stageThirdPartyPlugins({projectRoot: root, vendorDir})[0].action, 'unchanged');
  fs.writeFileSync(path.join(source, 'lib', 'api.js'), 'export const x = 1;\n');
  assert.equal(stageThirdPartyPlugins({projectRoot: root, vendorDir})[0].action, 'replaced');
  assert.ok(fs.statSync(path.join(vendorDir, 'dsh-deepseek-tools', 'lib', 'api.js')).isFile());
  assert.deepEqual(fs.readdirSync(vendorDir), ['dsh-deepseek-tools'], 'no staging leftovers');
});

test('a missing or incomplete source refuses instead of staging half a plugin', () => {
  const {root, source, vendorDir} = fixture();
  fs.rmSync(path.join(source, 'lib', 'index.js'));
  assert.throws(() => stageThirdPartyPlugins({projectRoot: root, vendorDir}), /source package is incomplete: lib\/index\.js/);
  fs.rmSync(source, {recursive: true, force: true});
  assert.throws(() => stageThirdPartyPlugins({projectRoot: root, vendorDir}), /source package\.json is missing/);
  assert.deepEqual(fs.readdirSync(vendorDir), [], 'nothing was staged');
});

test('a runtime that deliberately ships without paid application APIs says so', () => {
  const {root, vendorDir} = fixture();
  const staged = stageThirdPartyPlugins({projectRoot: root, vendorDir, without: ['dsh-deepseek-tools']});
  assert.equal(staged[0].action, 'skipped');
  assert.deepEqual(fs.readdirSync(vendorDir), []);
});

test('the manifest and every exported entry have to exist', () => {
  const {source} = fixture();
  assert.equal(verifyPluginSource(source, 'dsh-deepseek-tools').name, 'dsh-deepseek-tools');
  fs.rmSync(path.join(source, 'client', 'client.js'));
  assert.throws(() => verifyPluginSource(source, 'dsh-deepseek-tools'), /client\/client\.js \(exports \.\/client\)/);
  assert.throws(() => verifyPluginSource(source, 'other-name'), /names "dsh-deepseek-tools"/);
});

test("leftovers from an interrupted stage do not accumulate", () => {
  const {root, source, vendorDir} = fixture();
  stagePlugin({source, vendorDir, name: 'dsh-deepseek-tools'});
  fs.mkdirSync(path.join(vendorDir, ".dsh-deepseek-tools.previous"), {recursive: true});
  fs.mkdirSync(path.join(vendorDir, ".dsh-deepseek-tools.staged"), {recursive: true});
  fs.writeFileSync(path.join(source, "lib", "extra.js"), String.raw`export {};` + String.fromCharCode(10));
  assert.equal(stageThirdPartyPlugins({projectRoot: root, vendorDir})[0].action, "replaced");
  assert.deepEqual(fs.readdirSync(vendorDir), ["dsh-deepseek-tools"]);
  assert.ok(fs.statSync(path.join(vendorDir, "dsh-deepseek-tools", "lib", "extra.js")).isFile());
});
