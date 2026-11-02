'use strict';
/*
 * Native desktop-pet idle behaviour, adapted from ChisaAlter/WhaleIsle
 * commit 90465b67ec16765631d54c195af19d9ff3c3a770:
 * src/main/pet-settings.js, src/renderer/pet-live2d.js and pet-dialogue.js.
 *
 * MIT License
 * 
 * Copyright (c) 2026 Chisa
 * 
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 * 
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 * 
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */
(function () {
  const TALK_MS = {
    quiet: [360000, 720000],
    balanced: [120000, 240000],
    active: [40000, 80000],
  };
  const PERSONAS = ['natural', 'genki', 'tsundere', 'poison'];
  const CLINGY_BIAS = { natural: 0, genki: 0.15, tsundere: -0.15, poison: -0.05 };
  const IDLE_TOPICS = ['idle', 'idleRice', 'idleStandby', 'idleTail', 'idleCoding', 'idleCare'];
  const TIME_RANGES = {
    latenight: [23, 6], morning: [6, 11], noon: [11, 14],
    afternoon: [14, 18], evening: [18, 23],
  };
  const CATEGORIES = [...IDLE_TOPICS, ...Object.keys(TIME_RANGES),
    'sleepy', 'hungry', 'clingy', 'personalitySet'];
  const ACTION_SELECTOR = 'a,button,input,textarea,select,[role="button"],[contenteditable="true"],[data-pet-action]';
  const BODY_SELECTOR = '#sprite,#fallback';
  const DEFAULTS = { personality: 'natural', activity: 'balanced', selfTalk: true, powerSave: false };
  const own = (object, key) => Object.prototype.hasOwnProperty.call(object, key);
  const isObject = value => !!value && typeof value === 'object' && !Array.isArray(value);

  function normalized(raw, previous) {
    const value = isObject(raw) ? raw : {};
    return {
      personality: PERSONAS.includes(value.personality) ? value.personality : previous.personality,
      activity: own(TALK_MS, value.activity) ? value.activity : previous.activity,
      selfTalk: typeof value.selfTalk === 'boolean' ? value.selfTalk : previous.selfTalk,
      powerSave: typeof value.powerSave === 'boolean' ? value.powerSave : previous.powerSave,
    };
  }

  function cleanPools(raw) {
    const result = Object.create(null);
    if (!isObject(raw)) return result;
    for (const key of CATEGORIES) {
      const pool = raw[key];
      if (!Array.isArray(pool)) continue;
      const lines = pool.filter(line => typeof line === 'string' && line.trim());
      if (lines.length) result[key] = lines;
    }
    return result;
  }

  function cleanStore(raw) {
    if (!isObject(raw)) return null;
    const result = { global: cleanPools(raw.global), agents: Object.create(null),
      idleTopics: IDLE_TOPICS.slice(), timeOfDay: { ...TIME_RANGES } };
    for (const persona of PERSONAS.slice(1)) {
      result.agents[persona] = cleanPools(isObject(raw.agents) ? raw.agents[persona] : null);
    }
    if (Array.isArray(raw.idleTopics)) {
      const topics = raw.idleTopics.filter(topic => IDLE_TOPICS.includes(topic));
      if (topics.length) result.idleTopics = [...new Set(topics)];
    }
    if (isObject(raw.timeOfDay)) {
      for (const category of Object.keys(TIME_RANGES)) {
        const range = raw.timeOfDay[category];
        if (Array.isArray(range) && range.length === 2
            && range.every(hour => Number.isInteger(hour) && hour >= 0 && hour <= 24)) {
          result.timeOfDay[category] = range.slice();
        }
      }
    }
    return result;
  }

  // bubble is the shared DOM element (initially hidden). Status has the
  // task.json shape; Care.Satiety and Care.AffectionLevel are numeric.
  // now and touch use performance.now(), the same clock as requestAnimationFrame.
  // Call applySettings after each status/settings update, and touch on native
  // interactions not represented by DOM events. No Host/model bridge is used.
  function create({ bubble, getSettings, getStatus, isSleeping, body } = {}) {
    let settings = { ...DEFAULTS };
    try { settings = normalized(typeof getSettings === 'function' ? getSettings() : null, settings); } catch {}
    let closed = false, store = null, talkTimer = null, bubbleTimer = null;
    let lastInteraction = performance.now(), hoverTarget = null, ownedText = '';
    const bags = new Map(), lastSaid = new Map();
    const controller = new AbortController();

    function closest(target, selector) {
      return target && target.isConnected !== false && typeof target.closest === 'function'
        ? target.closest(selector) : null;
    }

    function interactive(target) {
      return !!closest(target, ACTION_SELECTOR)
        || (body ? body.matches(target) : !!closest(target, BODY_SELECTOR));
    }

    function interacting() {
      return interactive(hoverTarget) || interactive(document.activeElement);
    }

    function powerSaving(now) {
      const time = Number.isFinite(now) ? now : performance.now();
      return !closed && settings.powerSave && !interacting()
        && time - lastInteraction > 300000;
    }

    function readStatus() {
      try {
        const status = typeof getStatus === 'function' ? getStatus() : {};
        return isObject(status) ? status : {};
      } catch { return null; }
    }

    function sleeping(status) {
      if (status && status.Rest && status.Rest.Sleeping === true) return true;
      try { return typeof isSleeping === 'function' && !!isSleeping(); }
      catch { return true; }
    }

    function occupied(status) {
      return !status || sleeping(status) || status.QuickChatOpen === true || ['Care', 'Chat', 'Look', 'Growth']
        .some(key => status[key] && status[key].Busy === true);
    }

    function actionInteraction() {
      return !!closest(hoverTarget, ACTION_SELECTOR)
        || !!closest(document.activeElement, ACTION_SELECTOR);
    }

    function clearOwnBubble() {
      clearTimeout(bubbleTimer); bubbleTimer = null;
      // An external event may have replaced the text. Never clear its bubble.
      if (ownedText && bubble && bubble.textContent === ownedText) {
        bubble.hidden = true;
        bubble.textContent = '';
      }
      ownedText = '';
    }

    function touch() {
      if (closed) return;
      lastInteraction = performance.now();
      clearOwnBubble();
    }

    function bubbleVisible() {
      return bubble && !bubble.hidden
        && (!!bubble.textContent.trim() || bubble.childElementCount > 0);
    }

    function suppressed(now, status) {
      return closed || !bubble || document.hidden || occupied(status)
        || actionInteraction() || bubbleVisible() || powerSaving(now)
        || now - lastInteraction < 4000;
    }

    function phrase(category) {
      if (!store || !CATEGORIES.includes(category)) return '';
      const persona = settings.personality;
      const overrides = store.agents[persona];
      const pool = (overrides && overrides[category]) || store.global[category];
      if (!Array.isArray(pool) || !pool.length) return '';
      const key = persona + ':' + category;
      let bag = bags.get(key);
      if (!bag || !bag.length) {
        bag = pool.map((_, index) => index);
        for (let i = bag.length - 1; i > 0; i -= 1) {
          const j = Math.floor(Math.random() * (i + 1));
          [bag[i], bag[j]] = [bag[j], bag[i]];
        }
        if (bag.length > 1 && bag[bag.length - 1] === lastSaid.get(key)) {
          [bag[0], bag[bag.length - 1]] = [bag[bag.length - 1], bag[0]];
        }
        bags.set(key, bag);
      }
      const index = bag.pop();
      lastSaid.set(key, index);
      return pool[index];
    }

    function speak(category, now, status) {
      if (suppressed(now, status)) return;
      const text = phrase(category);
      if (!text) return;
      ownedText = text;
      bubble.textContent = text; // Dialogue is data, never markup.
      bubble.hidden = false;
      clearTimeout(bubbleTimer);
      bubbleTimer = setTimeout(clearOwnBubble,
        Math.min(2200 + Array.from(text).length * 90, 5600));
    }

    function category(now, status) {
      // Upstream eases drowsiness toward one at 0.6/sec after 90 idle seconds.
      const sleepy = 1 - Math.exp(-0.6 * Math.max(0, now - lastInteraction - 90000) / 1000);
      const care = status.Care || {};
      if (sleepy > 0.5 && Math.random() < 0.4) return 'sleepy';
      if (Number.isFinite(care.Satiety) && care.Satiety < 20 && Math.random() < 0.55) return 'hungry';
      if (Number.isFinite(care.AffectionLevel) && care.AffectionLevel >= 4
          && Math.random() < 0.25 + CLINGY_BIAS[settings.personality]) return 'clingy';
      let timeCategory = 'idle';
      const hour = new Date().getHours();
      for (const [name, [start, end]] of Object.entries(store.timeOfDay)) {
        if (start <= end ? hour >= start && hour < end : hour >= start || hour < end) {
          timeCategory = name; break;
        }
      }
      return Math.random() < 0.45 ? timeCategory
        : store.idleTopics[Math.floor(Math.random() * store.idleTopics.length)];
    }

    function schedule(first) {
      clearTimeout(talkTimer); talkTimer = null;
      if (closed || !store || !settings.selfTalk || document.hidden) return;
      const [minimum, maximum] = TALK_MS[settings.activity];
      const interval = minimum + Math.random() * (maximum - minimum);
      talkTimer = setTimeout(talk, first ? Math.min(interval, 90000) : interval);
    }

    function talk() {
      talkTimer = null;
      if (closed) return;
      const now = performance.now(), status = readStatus();
      // Missed/blocked slots are dropped, never queued or replayed on wake.
      try {
        if (settings.selfTalk && store && !suppressed(now, status)) {
          speak(category(now, status), now, status);
        }
      } finally { schedule(false); }
    }

    function applySettings(next) {
      if (closed) return;
      const previous = settings;
      settings = normalized(next, settings);
      if (!settings.selfTalk || document.hidden || occupied(readStatus())) clearOwnBubble();
      if (settings.activity !== previous.activity || settings.selfTalk !== previous.selfTalk) schedule(true);
      if (settings.personality !== previous.personality) {
        speak('personalitySet', performance.now(), readStatus());
      }
    }

    function onPointerOver(event) {
      hoverTarget = event.target;
      if (interactive(hoverTarget)) touch();
    }
    function onPointerOut(event) {
      hoverTarget = event.relatedTarget;
      if (interactive(event.target)) touch();
    }
    function onFocus() { touch(); }
    function onVisibility() {
      clearOwnBubble();
      // Visibility restoration gets a fresh slot, not an overdue burst.
      schedule(false);
    }

    function close() {
      if (closed) return;
      closed = true;
      controller.abort();
      clearTimeout(talkTimer); talkTimer = null;
      clearOwnBubble();
      bags.clear(); lastSaid.clear(); store = null; hoverTarget = null;
      document.removeEventListener('pointerover', onPointerOver);
      document.removeEventListener('pointerout', onPointerOut);
      document.removeEventListener('pointerdown', touch);
      document.removeEventListener('focusin', onFocus);
      document.removeEventListener('focusout', onFocus);
      document.removeEventListener('keydown', touch);
      document.removeEventListener('visibilitychange', onVisibility);
      window.removeEventListener('pagehide', close);
    }

    document.addEventListener('pointerover', onPointerOver);
    document.addEventListener('pointerout', onPointerOut);
    document.addEventListener('pointerdown', touch);
    document.addEventListener('focusin', onFocus);
    document.addEventListener('focusout', onFocus);
    document.addEventListener('keydown', touch);
    document.addEventListener('visibilitychange', onVisibility);
    window.addEventListener('pagehide', close, { once: true });
    // Fixed packaged data only. Redirects and cross-origin fetches are refused.
    void fetch(new URL('pet-dialogue.json', window.location.href), {
      mode: 'same-origin', credentials: 'omit', redirect: 'error', signal: controller.signal,
    }).then(response => {
      if (!response.ok) throw new Error('pet dialogue unavailable');
      return response.json();
    }).then(raw => {
      if (closed) return;
      store = cleanStore(raw);
      schedule(true);
    }).catch(() => { /* Optional local chatter must never break the pet. */ });

    return Object.freeze({ applySettings, touch, powerSaving, close });
  }

  window.WhalePetBehavior = Object.freeze({ create });
})();

