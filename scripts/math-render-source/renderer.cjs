'use strict';
const {mathjax} = require('mathjax-full/js/mathjax.js');
const {TeX} = require('mathjax-full/js/input/tex.js');
const {SVG} = require('mathjax-full/js/output/svg.js');
const {liteAdaptor} = require('mathjax-full/js/adaptors/liteAdaptor.js');
const {RegisterHTMLHandler} = require('mathjax-full/js/handlers/html.js');
const {AllPackages} = require('mathjax-full/js/input/tex/AllPackages.js');
const adaptor = liteAdaptor();
RegisterHTMLHandler(adaptor);
// Formula data never loads code, HTML links, files or network resources. Every
// supported package is bundled ahead of time; runtime package loading is off.
const packages = AllPackages.filter(name => !['autoload', 'require', 'html', 'noerrors', 'noundefined'].includes(name));
function render(request) {
  try {
    if (!request || typeof request.tex !== 'string' || Buffer.byteLength(request.tex, 'utf8') > 32768) throw Error('Formula exceeds 32 KiB');
    const input = new TeX({packages, maxBuffer: 32768, maxMacros: 1000, formatError: (_jax, error) => { throw error; }});
    const output = new SVG({fontCache: 'none', internalSpeechTitles: false});
    const document = mathjax.document('', {InputJax: input, OutputJax: output});
    const container = document.convert(request.tex, {display: request.display === true, em: 16, ex: 8, containerWidth: 80 * 16});
    const svg = adaptor.firstChild(container);
    const box = adaptor.getAttribute(svg, 'viewBox').split(/\s+/).map(Number);
    if (box.length !== 4 || box.some(x => !Number.isFinite(x)) || box[2] < 0 || box[3] <= 0) throw Error('Invalid equation dimensions');
    const widthEm = Math.max(0.01, box[2] / 1000), heightEm = box[3] / 1000, depthEm = Math.max(0, (box[1] + box[3]) / 1000);
    adaptor.setAttribute(svg, 'width', String(widthEm));
    adaptor.setAttribute(svg, 'height', String(heightEm));
    adaptor.removeAttribute(svg, 'style');
    const source = adaptor.outerHTML(svg);
    // MyGo SVG does not implement text or foreignObject. Do not silently lose
    // glyphs that MathJax's bundled font cannot turn into paths.
    if (/<(?:text|foreignObject|image|script)\b/i.test(source)) throw Error('Formula contains a glyph without an embedded vector outline');
    if (Buffer.byteLength(source, 'utf8') > 4 * 1024 * 1024) throw Error('Equation output exceeds 4 MiB');
    return {svg: source, widthEm, heightEm, depthEm};
  } catch (error) {
    return {error: String(error.message || error).slice(0, 512)};
  }
}
globalThis.whaleisleRenderMath = render;
