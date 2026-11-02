'use strict';
// Renderer geometry is page-local. Native Go still owns the stage, movement,
// display choice and the single 8-DIP native hover margin.
(function (root) {
  function create({characters = [], target = null} = {}) {
    const elements = Object.freeze(characters.filter(Boolean));
    const targets = Object.freeze(target ? [...elements, target] : [...elements]);
    const captures = new Map(), released = new Set();
    let owner = null, presentation = null, reserved = null, captureVersion = 0;
    const containsPoint = (r, x, y) => x >= r.left && x <= r.right && y >= r.top && y <= r.bottom;
    function padded(r) { return {x:r.x-8,y:r.y-8,left:r.x-8,top:r.y-8,right:r.x+r.width+8,bottom:r.y+r.height+8,width:r.width+16,height:r.height+16}; }
    function active() { return elements.find(element => !element.hidden) || elements[0] || null; }
    function rect(element = active()) {
      if (presentation && (element === presentation.visual || element === target)) return padded(presentation.rect);
      return element ? element.getBoundingClientRect() : null;
    }
    function hitRect(element) {
      if (presentation && element === presentation.visual) return null;
      if (element === target) return presentation?.rect || null;
      return rect(element);
    }
    function contains(x, y) {
      if (presentation) return containsPoint(padded(presentation.rect), x, y);
      return elements.some(element => !element.hidden && containsPoint(rect(element), x, y));
    }
    function matches(node) {
      const interactive = presentation && target ? [target] : elements;
      return !!node && node.isConnected !== false && interactive.some(element => element === node || element.contains?.(node));
    }
    function capture(element, id, held) {
      if (held) captures.set(id, element); else if (captures.get(id) === element) captures.delete(id);
      captureVersion++;
      if (!held && !captures.size) for (const callback of released) callback();
    }
    function acquire(visual) {
      const handle = {};
      owner = handle;
      return Object.freeze({
        publish(value) { if (owner !== handle) return false; reserved = visual; presentation = {visual,rect:value}; return true; },
        reserve(value) { if (owner === handle) reserved = value ? visual : null; },
        clear() { if (owner !== handle) return false; presentation = null; reserved = null; return true; },
        close() { if (owner !== handle) return; presentation = null; reserved = null; owner = null; },
        current() { return owner === handle; },
      });
    }
    return Object.freeze({elements, targets, target, active, rect, hitRect, contains, matches, acquire, capture,
      capturing(element) { return element ? [...captures.values()].includes(element) : captures.size > 0; },
      onRelease(callback) { released.add(callback); return () => released.delete(callback); },
      get captureVersion() { return captureVersion; },
      managed(element) { return reserved === element; },
      hoverSelector: target ? '#sprite:hover,#fallback:hover,#pet-body-hit:hover' : '#sprite:hover,#fallback:hover',
      // Native nominal anchors remain unchanged in this phase.
      anchorRect(layout) { return layout; }});
  }
  root.WhalePetBody = Object.freeze({create, current: create({characters: [
    document.getElementById('sprite'), document.getElementById('fallback'),
  ], target:document.getElementById('pet-body-hit')})});
})(window);
