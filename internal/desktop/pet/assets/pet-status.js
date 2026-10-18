'use strict';
(function () {
  const card = document.getElementById('pet-status-card');
  const toggle = document.getElementById('pet-status-toggle');
  const close = document.getElementById('pet-status-close');
  function show(open) {
    card.hidden = !open;
    toggle.setAttribute('aria-expanded', String(open));
    if (!open && card.contains(document.activeElement)) toggle.focus();
  }
  toggle.addEventListener('click', () => show(card.hidden));
  close.addEventListener('click', () => show(false));
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape' && !card.hidden) { event.preventDefault(); show(false); }
  });
  document.addEventListener('pointerdown', event => {
    if (!card.hidden && !card.contains(event.target) && !toggle.contains(event.target)) show(false);
  });
})();
