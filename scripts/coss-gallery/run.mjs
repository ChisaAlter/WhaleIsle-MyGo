// Runs the coss component gallery end to end and leaves only the report:
//
//   node scripts/coss-gallery/run.mjs [--out tmp/coss-gallery] [--only button,input] [--keep]
//
// 1. go test ./internal/ui/coss -run TestGalleryRender renders every
//    specimen in light and dark, headless, into <out>/run/native.
// 2. ref.mjs renders coss's own markup for the same specimens in headless
//    Chromium (playwright-core + an installed chrome-headless-shell; nothing
//    is downloaded but the npm packages, through the shared npm cache).
// 3. go test -run TestGalleryDiff writes <out>/run/report.md and the
//    side-by-side sheets.
// The npm packages, globals.css and the full run live in a work directory
// this script owns, removed in finally unless --keep; <out> keeps report.md
// and the light side-by-sides of a few components.
import { execFileSync, execSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';

const repo = path.resolve(path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')), '../..');
const args = process.argv.slice(2);
const opt = (name, def) => {
  const i = args.indexOf(name);
  return i >= 0 ? args[i + 1] : def;
};
const out = path.resolve(repo, opt('--out', 'tmp/coss-gallery'));
const only = opt('--only', '');
const keep = args.includes('--keep');
// --globals reuses a local copy of coss's globals.css instead of fetching it.
const globalsArg = opt('--globals', '');
const SIDES = ['button', 'input', 'check', 'popup', 'dialog', 'layout', 'chat', 'composer'];
const GLOBALS = 'https://raw.githubusercontent.com/cosscom/ui/b2336c45f2b7355c42678ce786b6f14d62b819be/packages/ui/src/styles/globals.css';

fs.mkdirSync(out, { recursive: true });
const work = fs.mkdtempSync(path.join(out, '.work-'));
const run = path.join(work, 'run');
function run_(cmd, argv, opts = {}) {
  execFileSync(cmd, argv, { stdio: 'inherit', ...opts });
}
try {
  const env = { ...process.env, COSS_GALLERY_DIR: run, COSS_GALLERY_ONLY: only };
  run_('go', ['test', '-count=1', '-run', '^TestGalleryRender$', './internal/ui/coss'], { cwd: repo, env });
  fs.writeFileSync(path.join(work, 'package.json'), '{"private":true,"type":"module"}');
  execSync('npm i --no-audit --no-fund --silent tailwindcss@4.1.17 tailwind-merge@3 playwright-core@1.65.0', {
    cwd: work, stdio: 'inherit', env: { ...process.env, PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: '1' },
  });
  if (globalsArg) {
    fs.copyFileSync(path.resolve(globalsArg), path.join(work, 'globals.css'));
  } else {
    let text = '';
    for (let i = 0; i < 3 && !text; i++) {
      try {
        const res = await fetch(GLOBALS);
        if (res.ok) text = await res.text();
      } catch (err) {
        if (i === 2) throw err;
      }
    }
    if (!text) throw new Error('globals.css could not be fetched; pass --globals <file>');
    fs.writeFileSync(path.join(work, 'globals.css'), text);
  }
  run_(process.execPath, [path.join(repo, 'scripts/coss-gallery/ref.mjs'), run, path.join(work, 'globals.css')], { cwd: work });
  run_('go', ['test', '-count=1', '-run', '^TestGalleryDiff$', './internal/ui/coss'], { cwd: repo, env });
  fs.copyFileSync(path.join(run, 'report.md'), path.join(out, 'report.md'));
  fs.mkdirSync(path.join(out, 'side'), { recursive: true });
  for (const comp of SIDES) {
    const f = path.join(run, 'side', comp + '-light.png');
    if (fs.existsSync(f)) fs.copyFileSync(f, path.join(out, 'side', comp + '-light.png'));
  }
  console.log('report: ' + path.join(out, 'report.md'));
} finally {
  if (keep) {
    console.log('kept: ' + work);
  } else {
    // Only the directory this run created, never through a junction.
    const real = fs.realpathSync(work);
    if (path.dirname(real) === fs.realpathSync(out) && path.basename(real).startsWith('.work-')) {
      fs.rmSync(real, { recursive: true, force: true });
    }
  }
}
