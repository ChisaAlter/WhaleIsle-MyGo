'use strict';
(function () {
  const body = window.WhalePetBody?.current;
  const characters = body?.elements || [document.getElementById('sprite'), document.getElementById('fallback')];
  const overPet = event => body ? body.contains(event.clientX, event.clientY) : characters.some(el => {
    if (!el || el.hidden) return false;
    const r = el.getBoundingClientRect();
    return event.clientX >= r.left && event.clientX <= r.right && event.clientY >= r.top && event.clientY <= r.bottom;
  });
  document.addEventListener('dragover', event => {
    if (Array.from(event.dataTransfer?.types || []).includes('Files') && overPet(event)) {
      event.preventDefault(); event.dataTransfer.dropEffect = 'copy';
    }
  });
  document.addEventListener('drop', event => {
    if (!event.dataTransfer?.files?.length || !overPet(event)) return;
    event.preventDefault();
    // File metadata only: never read bytes, paths, or upload a dropped file.
    const name = String(event.dataTransfer.files[0].name || '文件').slice(0, 240);
    location.href = 'whale-pet://localhost/feed-drop?name=' + encodeURIComponent(name);
  });
})();
