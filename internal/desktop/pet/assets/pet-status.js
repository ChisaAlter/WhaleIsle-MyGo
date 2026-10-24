'use strict';
// The native pet has no permanent chrome at rest: pet-live2d.html:275-303 keeps
// only the #pet canvas visible, and the overlay's actions live behind a
// right-click (context menu) plus the tray. This page mirrors that: the status
// card is closed by default, the toggle appears on hover or keyboard focus, and
// a right-click on the pet opens the same card (Escape and outside clicks close
// it). Every action inside the card is unchanged.
(function () {
  const card = document.getElementById('pet-status-card');
  const toggle = document.getElementById('pet-status-toggle');
  const close = document.getElementById('pet-status-close');
  const sprite = document.getElementById('sprite');
  const fallback = document.getElementById('fallback');

  function show(open) {
    card.hidden = !open;
    toggle.setAttribute('aria-expanded', String(open));
    if (open) {
      const first = card.querySelector('a, button');
      if (first && document.activeElement === toggle) first.focus();
    } else if (card.contains(document.activeElement)) {
      toggle.focus();
    }
  }

  toggle.addEventListener('click', () => show(card.hidden));
  close.addEventListener('click', () => show(false));

  // Right-click on the pet is the native affordance for its menu.
  for (const target of [sprite, fallback]) {
    if (!target) continue;
    target.addEventListener('contextmenu', (event) => {
      event.preventDefault();
      show(true);
    });
  }
  // A long press keeps the card reachable without a right mouse button.
  for (const target of [sprite, fallback]) {
    if (!target) continue;
    let pressTimer = 0;
    target.addEventListener('pointerdown', () => {
      clearTimeout(pressTimer);
      pressTimer = setTimeout(() => show(true), 600);
    });
    for (const done of ['pointerup', 'pointerleave', 'pointercancel']) {
      target.addEventListener(done, () => clearTimeout(pressTimer));
    }
  }

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && !card.hidden) {
      event.preventDefault();
      show(false);
    }
  });
  document.addEventListener('pointerdown', (event) => {
    if (!card.hidden && !card.contains(event.target) && !toggle.contains(event.target)) show(false);
  });
})();
