(() => {
  // This is host-initiated DOM inspection only. No native message handler,
  // application binding, network request, input value or HTML is exposed.
  const key = __PICKER_KEY__;
  const state = { done: false, cancelled: false, element: null };
  let stopped = false, timer, overlay, shield;
  const clean = value => String(value == null ? '' : value)
    .replace(/[\u0000-\u001f\u007f]/g, ' ').slice(0, 512);
  const escape = value => {
    if (globalThis.CSS && typeof CSS.escape === 'function') return CSS.escape(value);
    return String(value).replace(/[^a-zA-Z0-9_-]/g, c => '\\' + c.codePointAt(0).toString(16) + ' ');
  };
  const selector = element => {
    const parts = [];
    for (let node = element; node && node.nodeType === 1 && parts.length < 6; node = node.parentElement) {
      let part = node.localName || 'element';
      if (node.id) {
        part += '#' + escape(node.id.slice(0, 128));
        parts.unshift(part);
        break;
      }
      if (node.parentElement) {
        let index = 1;
        for (let previous = node.previousElementSibling; previous; previous = previous.previousElementSibling) {
          if (previous.localName === node.localName) index++;
        }
        part += ':nth-of-type(' + index + ')';
      }
      parts.unshift(part);
    }
    return parts.join(' > ').slice(0, 1024);
  };
  const target = event => {
    // The transparent shield also makes cross-origin frames selectable as a
    // frame element; their private documents are never traversed.
    shield.style.pointerEvents = 'none';
    let element;
    try {
      element = document.elementFromPoint(event.clientX, event.clientY);
      for (let depth = 0; element && element.shadowRoot && depth < 8; depth++) {
        const nested = element.shadowRoot.elementFromPoint(event.clientX, event.clientY);
        if (!nested || nested === element) break;
        element = nested;
      }
    } finally { shield.style.pointerEvents = 'auto'; }
    return element instanceof Element && element !== overlay && element !== shield ? element : null;
  };
  const block = event => { event.preventDefault(); event.stopImmediatePropagation(); };
  const move = event => {
    const element = target(event);
    if (!element || !overlay) return;
    const rect = element.getBoundingClientRect();
    Object.assign(overlay.style, {
      left: rect.left + 'px', top: rect.top + 'px',
      width: Math.max(0, rect.width) + 'px', height: Math.max(0, rect.height) + 'px',
      display: 'block'
    });
  };
  const down = event => block(event);
  const finish = cancelled => {
    if (stopped) return;
    stopped = true;
    state.done = true;
    state.cancelled = cancelled;
    clearTimeout(timer);
    document.removeEventListener('pointermove', move, true);
    document.removeEventListener('pointerdown', down, true);
    document.removeEventListener('mousedown', down, true);
    document.removeEventListener('mouseup', down, true);
    document.removeEventListener('click', click, true);
    document.removeEventListener('keydown', keydown, true);
    globalThis.removeEventListener('pagehide', pagehide, true);
    if (overlay) overlay.remove();
    if (shield) shield.remove();
  };
  const click = event => {
    block(event);
    if (!event.isTrusted) return;
    const element = target(event);
    if (!element) return;
    const rect = element.getBoundingClientRect();
    // Never inspect form values, editable text, password fields or whole HTML.
    const sensitive = element.matches('input, textarea, select, [contenteditable]') ||
      element.closest('input, textarea, select, [contenteditable]') ||
      element.querySelector('input, textarea, select, [contenteditable]');
    state.element = {
      tag: clean(element.localName), selector: selector(element),
      text: sensitive ? '' : clean(element.textContent),
      role: clean(element.getAttribute('role')),
      label: sensitive ? '' : clean(element.getAttribute('aria-label')),
      x: rect.left, y: rect.top, width: rect.width, height: rect.height
    };
    finish(false);
  };
  const keydown = event => {
    if (event.key === 'Escape') { block(event); finish(true); }
  };
  const pagehide = () => finish(true);
  shield = document.createElement('div');
  shield.setAttribute('aria-label', 'Choose an element; Escape cancels');
  Object.assign(shield.style, {
    position: 'fixed', inset: '0', zIndex: '2147483646',
    background: 'transparent', cursor: 'crosshair', pointerEvents: 'auto'
  });
  overlay = document.createElement('div');
  overlay.setAttribute('aria-hidden', 'true');
  Object.assign(overlay.style, {
    position: 'fixed', zIndex: '2147483647', pointerEvents: 'none',
    boxSizing: 'border-box', border: '2px solid #3b82f6',
    background: 'rgba(59,130,246,0.12)', display: 'none'
  });
  (document.documentElement || document.body).appendChild(shield);
  (document.documentElement || document.body).appendChild(overlay);
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: Object.freeze({
      read: () => state,
      cancel: () => { finish(true); delete globalThis[key]; }
    })
  });
  document.addEventListener('pointermove', move, true);
  document.addEventListener('pointerdown', down, true);
  document.addEventListener('mousedown', down, true);
  document.addEventListener('mouseup', down, true);
  document.addEventListener('click', click, true);
  document.addEventListener('keydown', keydown, true);
  globalThis.addEventListener('pagehide', pagehide, true);
  timer = setTimeout(() => finish(true), 60000);
  return true;
})()
