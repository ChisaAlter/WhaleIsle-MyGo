// Windows installer source/resource regression only. Never runs a packager,
// installer, application, Host, model, or network operation.
import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const script = fs.readFileSync(path.join(root, 'scripts/nsis/whaleisle.nsi'), 'utf8');
const branding = path.join(root, 'scripts/nsis/branding');
const source = JSON.parse(fs.readFileSync(path.join(branding, 'source-assets.json'), 'utf8'));
const sha = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
// Source: original package.json build.nsis and assistedInstaller.nsh. This
// protects the page order that was missing when only directory/instfiles existed.
const install = ['WELCOME', 'LICENSE', 'DIRECTORY', 'INSTFILES', 'FINISH'];
const uninstall = ['WELCOME', 'INSTFILES', 'FINISH'];
assert.deepEqual([...script.matchAll(/^!insertmacro MUI_PAGE_(\w+)/gm)].map(m => m[1]), install);
assert.deepEqual([...script.matchAll(/^!insertmacro MUI_UNPAGE_(\w+)/gm)].map(m => m[1]), uninstall);
assert.deepEqual([...script.matchAll(/^!insertmacro MUI_LANGUAGE "([^"]+)"/gm)].map(m => m[1]), ['SimpChinese', 'English']);
assert.match(script, /^Unicode true$/m);
assert.match(script, /^!include "MUI2\.nsh"$/m);
assert.doesNotMatch(script, /^\s*(?:Page|UninstPage)\s/m);
// The original does not opt into displayLanguageSelector. MUI chooses the OS
// language; adding a selector would insert an unrequested extra user step.
assert.doesNotMatch(script, /^\s*!insertmacro MUI_LANGDLL_DISPLAY/m);
for (const [define, resource] of [
  ['MUI_ICON', 'icon.ico'], ['MUI_UNICON', 'icon.ico'],
  ['MUI_HEADERIMAGE_BITMAP', 'installerHeader.bmp'],
  ['MUI_WELCOMEFINISHPAGE_BITMAP', 'installerSidebar.bmp'],
  ['MUI_UNWELCOMEFINISHPAGE_BITMAP', 'uninstallerSidebar.bmp'],
]) {
  assert.ok(script.includes(`!define ${define} "branding\\${resource}"`), define);
}
assert.match(script, /^!define MUI_HEADERIMAGE_RIGHT$/m);
assert.match(script, /^BrandingText "\$\{PRODUCT_NAME\} \$\{VERSION\}"$/m);
assert.match(script, /^!insertmacro MUI_PAGE_LICENSE "branding\\LICENSE"$/m);
assert.match(script, /^!define MUI_FINISHPAGE_RUN "\$INSTDIR\\\$\{EXE_NAME\}"$/m);
assert.match(script, /^!define MUI_FINISHPAGE_LINK_LOCATION "https:\/\/github\.com\/ChisaAlter\/Deepseek-Harness-Desktop"$/m);
// MUI clears per-page defines: install and uninstall welcome must each set it.
assert.match(script, /!define MUI_WELCOMEPAGE_TITLE_3LINES\s+!insertmacro MUI_PAGE_WELCOME/);
assert.match(script, /!define MUI_WELCOMEPAGE_TITLE_3LINES\s+!insertmacro MUI_UNPAGE_WELCOME/);
assert.match(script, /!define MUI_FINISHPAGE_TITLE_3LINES[\s\S]*?!insertmacro MUI_PAGE_FINISH/);
assert.equal(source.repository, 'https://github.com/ChisaAlter/WhaleIsle');
assert.deepEqual(source.files.map(f => f.file).sort(), ['LICENSE', 'icon.ico', 'installerHeader.bmp', 'installerSidebar.bmp', 'uninstallerSidebar.bmp']);
const resources = [];
for (const entry of source.files) {
  const bytes = fs.readFileSync(path.join(branding, entry.file));
  assert.equal(bytes.length, entry.size, `${entry.file} size`);
  assert.equal(sha(bytes), entry.sha256, `${entry.file} original bytes`);
  if (entry.file.endsWith('.bmp')) {
    assert.equal(bytes.toString('ascii', 0, 2), 'BM');
    assert.equal(bytes.readUInt16LE(28), 24, `${entry.file} NSIS RGB bitmap`);
    const dimensions = [bytes.readInt32LE(18), bytes.readInt32LE(22)];
    assert.deepEqual(dimensions, entry.file === 'installerHeader.bmp' ? [150, 57] : [164, 314]);
    resources.push({file: entry.file, dimensions, sha256: entry.sha256});
  } else if (entry.file === 'icon.ico') {
    assert.equal(bytes.readUInt16LE(0), 0);
    assert.equal(bytes.readUInt16LE(2), 1);
    assert.ok(bytes.readUInt16LE(4) > 0, 'ICO contains image frames');
    resources.push({file: entry.file, sha256: entry.sha256});
  } else {
    assert.match(bytes.toString('utf8'), /^MIT License/);
    assert.match(bytes.toString('utf8'), /Copyright \(c\) 2026 Chisa/);
    resources.push({file: entry.file, sha256: entry.sha256});
  }
}
console.log(JSON.stringify({
  status: 'PASS_SOURCE_AND_RESOURCE_CONTRACT_ONLY',
  script: 'scripts/nsis/whaleisle.nsi', install, uninstall,
  languages: ['zh_CN', 'en_US'], languageSelector: false, resources,
  nsisCompilation: 'NOT_EXECUTED_NO_LOCAL_NSIS',
  installation: 'NOT_EXECUTED_SYSTEM_CHANGES_PROHIBITED',
}, null, 2));
