'use strict';
// Native Go owns desktop coordinates. This layer only maps its local DIP layout
// into a transparent viewport; a page can never choose a display or move it.
(function (root) {
  function anchor(rect, size, area, gap = 8) {
    const left = area.X + 4, top = area.Y + 4;
    const right = Math.max(left, area.X + area.Width - size.width - 4);
    const bottom = Math.max(top, area.Y + area.Height - size.height - 4);
    const x = Math.min(right, Math.max(left, rect.X + rect.Width / 2 - size.width / 2));
    let y = rect.Y - size.height - gap;
    if (y < top) y = rect.Y + rect.Height + gap;
    return {x: Math.round(x), y: Math.round(Math.min(bottom, Math.max(top, y)))};
  }
  function create({body: petBody} = {}) {
    let layout = null;
    const body = document.body;
    function placeSurfaces() {
      if (!layout) return;
      const area = layout.Area?.Width > 0 ? layout.Area : {X:0,Y:0,Width:innerWidth,Height:innerHeight};
      for (const id of ['pet-status-card', 'pet-bubble', 'render-notice']) {
        const node = document.getElementById(id);
        if (!node || node.hidden || !node.textContent.trim()) continue;
        node.style.maxWidth = Math.max(0, area.Width - 8) + 'px';
        node.style.maxHeight = Math.max(0, area.Height - 8) + 'px';
        const size = node.getBoundingClientRect();
        const point = anchor(petBody ? petBody.anchorRect(layout) : layout, size, area);
        node.style.left = point.x + 'px'; node.style.top = point.y + 'px';
      }
    }
    function update(next) {
      if (!next || !(next.Width > 0) || !(next.Height > 0)) return;
      layout = next;
      for (const [key, value] of Object.entries({x:next.X,y:next.Y,width:next.Width,height:next.Height})) {
        body.style.setProperty('--pet-' + key, value + 'px');
      }
      placeSurfaces();
    }
    const observer = new ResizeObserver(placeSurfaces);
    for (const id of ['pet-status-card', 'pet-bubble', 'render-notice']) {
      const node = document.getElementById(id); if (node) observer.observe(node);
    }
    const visibility = new MutationObserver(placeSurfaces);
    visibility.observe(body, {subtree:true,attributes:true,attributeFilter:['hidden']});
    try { update(JSON.parse(body.dataset.layout || '{}')); } catch {}
    root.addEventListener('pagehide', () => {observer.disconnect();visibility.disconnect();}, {once:true});
    return {update};
  }
  root.WhalePetLayout = {anchor, create};
})(window);
