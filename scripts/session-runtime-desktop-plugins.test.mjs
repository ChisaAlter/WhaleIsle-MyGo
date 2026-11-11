import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {copyDesktopPackage} from './session-runtime-desktop-plugins.mjs';

test('a package without files retains its implementation and relative imports', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'mygo-plugin-copy-'));
  const source = path.join(root, 'source'), destination = path.join(root, 'staged');
  fs.mkdirSync(path.join(source, 'lib'), {recursive: true});
  fs.writeFileSync(path.join(source, 'package.json'), JSON.stringify({type:'module', main:'lib/index.js'}));
  fs.writeFileSync(path.join(source, 'lib/index.js'), "export {value} from './helper.js';");
  fs.writeFileSync(path.join(source, 'lib/helper.js'), 'export const value = 42;');
  copyDesktopPackage(source, destination);
  assert.equal((await import(pathToFileURL(path.join(destination, 'lib/index.js')))).value, 42);
});

test('an allowlist that omits its declared entry fails before swapping the installed plugin', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'mygo-plugin-entry-'));
  const source = path.join(root, 'source'), destination = path.join(root, 'prepared');
  fs.mkdirSync(source);
  fs.writeFileSync(path.join(source, 'package.json'), JSON.stringify({type:'module', main:'lib/index.js', files:[]}));
  assert.throws(() => copyDesktopPackage(source, destination), /ENOENT|Missing staged plugin entry/);
});
