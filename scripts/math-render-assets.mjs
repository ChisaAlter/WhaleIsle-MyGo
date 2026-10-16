// Build-time only: npm ci --ignore-scripts in scripts/math-render-source first.
// Runtime rendering is fully offline and needs only the product's bundled Node.
import {createRequire} from 'node:module';
import {readFile, writeFile, mkdir} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import {createHash} from 'node:crypto';
const input = new URL('./math-render-source/', import.meta.url);
const require = createRequire(new URL('package.json', input));
const {build} = require('esbuild');
const output = new URL('../internal/mathrender/', import.meta.url);
await mkdir(output, {recursive:true});
const result = await build({entryPoints:[fileURLToPath(new URL('renderer.cjs', input))], bundle:true, platform:'node', target:'node22', format:'cjs', minify:true, metafile:true, write:false, legalComments:'inline'});
const bundle = result.outputFiles[0].contents;
await writeFile(new URL('renderer.cjs', output), bundle);
const lock = JSON.parse(await readFile(new URL('package-lock.json', input), 'utf8'));
const bundled = Object.keys(result.metafile.inputs);
const packages = Object.entries(lock.packages).filter(([name,p]) => name && !p.dev && bundled.some(file => file.includes('/' + name + '/')));
let licenses = '# Offline math renderer third-party licenses\n\n';
const provenance = {engine:'MathJax', version:'3.2.2', source:'https://registry.npmjs.org/mathjax-full/-/mathjax-full-3.2.2.tgz', bundleSHA256:createHash('sha256').update(bundle).digest('hex'), packages:[]};
for (const [name,p] of packages) {
  let license;
  for (const filename of ['LICENSE', 'LICENSE.txt', 'LICENSE.md', 'license', 'LICENSE.MD']) {
    try {license = await readFile(new URL(name + '/' + filename, input), 'utf8'); break;} catch (error) {if (error.code !== 'ENOENT') throw error;}
  }
  if (!license) throw Error(`Missing license: ${name}`);
  licenses += `## ${name} ${p.version}\n\n${license}\n\n`;
  provenance.packages.push({name:name.replace(/^node_modules\//,''),version:p.version,resolved:p.resolved,integrity:p.integrity});
}
await writeFile(new URL('LICENSES.md', output), licenses);
await writeFile(new URL('provenance.json', output), JSON.stringify(provenance,null,2)+'\n');
console.log(`Built offline MathJax SVG renderer: ${bundle.length} bytes, ${provenance.bundleSHA256}`);
