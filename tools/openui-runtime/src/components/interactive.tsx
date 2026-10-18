// Independently authored, local-only interactions for the public OpenUI contracts.
import React, { createContext, useContext, useEffect, useId, useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { defineComponent, type DefinedComponent } from '@openuidev/react-lang';
import { z } from 'zod/v4';

const ink = 'var(--oui-ink, #172337)';
const muted = 'var(--oui-muted, #526379)';
const border = 'var(--oui-border, #d9e0e8)';
const surface = 'var(--oui-surface, #f3f6fa)';
const panel = 'var(--oui-panel, #fff)';
const buttonStyle: CSSProperties = { padding: '7px 11px', borderRadius: 7, border: `1px solid ${border}`, background: panel, color: ink, font: 'inherit', cursor: 'pointer' };
const focusableSelector = 'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';
const interactiveSelector = 'button, a, input, select, textarea, summary, [contenteditable="true"], [role="button"], [role="link"]';
const visuallyHidden: CSSProperties = { position: 'absolute', width: 1, height: 1, margin: -1, padding: 0, border: 0, overflow: 'hidden', clipPath: 'inset(50%)', whiteSpace: 'nowrap' };

function useReducedMotion() {
  const [reduced, setReduced] = useState(() => typeof window !== 'undefined' && typeof window.matchMedia === 'function' ? window.matchMedia('(prefers-reduced-motion: reduce)').matches : true);
  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return;
    const media = window.matchMedia('(prefers-reduced-motion: reduce)');
    const update = () => setReduced(media.matches);
    update();
    media.addEventListener('change', update);
    return () => media.removeEventListener('change', update);
  }, []);
  return reduced;
}

const PopoverAncestors = createContext<string[]>([]);

function PopoverView({ trigger, content }: { trigger: ReactNode; content: ReactNode }) {
  const id = useId();
  const ancestors = useContext(PopoverAncestors);
  const lineage = [...ancestors, id];
  const triggerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState({ left: 12, top: 12 });

  const close = (restoreFocus: boolean) => {
    setOpen(false);
    if (restoreFocus) triggerRef.current?.focus({ preventScroll: true });
  };

  useLayoutEffect(() => {
    if (!open || !triggerRef.current || !contentRef.current) return;
    const button = triggerRef.current;
    const popover = contentRef.current;
    const place = () => {
      const anchor = button.getBoundingClientRect();
      const box = popover.getBoundingClientRect();
      const width = document.documentElement.clientWidth;
      const height = document.documentElement.clientHeight;
      const left = Math.max(12, Math.min(anchor.left, width - box.width - 12));
      const below = anchor.bottom + 8;
      const top = below + box.height <= height - 12 ? below : Math.max(12, anchor.top - box.height - 8);
      setPosition((prior) => prior.left === left && prior.top === top ? prior : { left, top });
    };
    place();
    const first = popover.querySelector<HTMLElement>(focusableSelector);
    (first || popover).focus({ preventScroll: true });
    window.addEventListener('resize', place);
    window.addEventListener('scroll', place, true);
    const observer = typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(place);
    observer?.observe(popover);
    observer?.observe(button);
    return () => {
      window.removeEventListener('resize', place);
      window.removeEventListener('scroll', place, true);
      observer?.disconnect();
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const isInside = (target: EventTarget | null) => {
      if (!(target instanceof Node)) return false;
      if (triggerRef.current?.contains(target) || contentRef.current?.contains(target)) return true;
      const layer = target instanceof Element ? target.closest<HTMLElement>('[data-oui-popover-lineage]') : null;
      return layer?.dataset.ouiPopoverLineage?.split(' ').includes(id) ?? false;
    };
    const outsidePointer = (event: PointerEvent) => {
      if (isInside(event.target)) return;
      const targetIsFocusable = event.target instanceof Element && event.target.closest(`${focusableSelector}, [contenteditable="true"]`);
      const focusWasInPanel = contentRef.current?.contains(document.activeElement);
      setOpen(false);
      if (focusWasInPanel && !targetIsFocusable) triggerRef.current?.focus({ preventScroll: true });
    };
    const outsideFocus = (event: FocusEvent) => { if (!isInside(event.target)) setOpen(false); };
    const escape = (event: KeyboardEvent) => {
      const focused = document.activeElement;
      const owningLayer = focused instanceof Element ? focused.closest<HTMLElement>('[data-oui-popover-owner]') : null;
      const ownsFocus = triggerRef.current?.contains(focused) || owningLayer?.dataset.ouiPopoverOwner === id;
      if (event.key === 'Escape' && ownsFocus) {
        event.preventDefault();
        event.stopPropagation();
        setOpen(false);
        triggerRef.current?.focus({ preventScroll: true });
      }
    };
    document.addEventListener('pointerdown', outsidePointer, true);
    document.addEventListener('focusin', outsideFocus, true);
    document.addEventListener('keydown', escape, true);
    return () => {
      document.removeEventListener('pointerdown', outsidePointer, true);
      document.removeEventListener('focusin', outsideFocus, true);
      document.removeEventListener('keydown', escape, true);
    };
  }, [open, id]);

  return <>
    <div ref={triggerRef} role="button" tabIndex={0} aria-haspopup="dialog" aria-expanded={open} aria-controls={open ? id : undefined}
      onClick={() => open ? close(true) : setOpen(true)}
      onKeyDown={(event) => {
        if (event.target !== event.currentTarget) return;
        if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); open ? close(true) : setOpen(true); }
      }}
      style={{ ...buttonStyle, display: 'inline-flex', alignItems: 'center', gap: 6, maxWidth: '100%', textAlign: 'start' }}>{trigger}</div>
    {open && typeof document !== 'undefined' ? createPortal(
      <div ref={contentRef} id={id} data-oui-popover-lineage={lineage.join(' ')} data-oui-popover-owner={id} role="dialog" aria-label="Additional details" tabIndex={-1} className="oui-popover" style={{ position: 'fixed', zIndex: 1000, left: position.left, top: position.top, width: 'max-content', maxWidth: 'min(360px, calc(100vw - 24px))', maxHeight: 'min(70vh, calc(100vh - 24px))', overflow: 'auto', background: panel, color: ink, border: `1px solid ${border}`, borderRadius: 10, padding: 16, boxSizing: 'border-box', boxShadow: '0 8px 32px rgb(0 0 0 / 16%)', fontSize: 14, lineHeight: 1.5, overflowWrap: 'anywhere' }}>
        <PopoverAncestors.Provider value={lineage}><div style={{ minWidth: 0, whiteSpace: typeof content === 'string' ? 'pre-wrap' : undefined }}>{content}</div></PopoverAncestors.Provider>
        <button type="button" onClick={() => close(true)} style={{ ...buttonStyle, display: 'block', marginTop: 12 }}>Close</button>
      </div>, document.body,
    ) : null}
  </>;
}

const Popover = defineComponent({
  name: 'Popover', props: z.object({ trigger: z.any(), content: z.any() }),
  description: 'Local disclosure panel with Escape, outside dismissal, and focus restoration.',
  component: ({ props, renderNode }) => <PopoverView trigger={renderNode(props.trigger)} content={renderNode(props.content)} />,
});

function FlashcardView({ front, back, hint }: { front: ReactNode; back: ReactNode; hint?: string }) {
  const [flipped, setFlipped] = useState(false);
  const reducedMotion = useReducedMotion();
  const instructionId = useId();
  const faceStyle: CSSProperties = { gridArea: '1 / 1', minWidth: 0, padding: 20, borderRadius: 12, border: `1px solid ${border}`, background: panel, backfaceVisibility: 'hidden', WebkitBackfaceVisibility: 'hidden', boxSizing: 'border-box', display: 'flex', flexDirection: 'column', justifyContent: 'center', gap: 12, overflowWrap: 'anywhere', whiteSpace: 'pre-wrap' };
  return <section className="oui-flashcard" role="group" tabIndex={0} aria-label={`Flashcard, ${flipped ? 'back' : 'front'} side`} aria-describedby={instructionId} style={{ minWidth: 0, color: ink, perspective: 1200 }}
    onKeyDown={(event) => {
      if (event.target !== event.currentTarget) return;
      if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); setFlipped((value) => !value); }
    }}
    onClick={(event) => {
      const target = event.target;
      if (!(target instanceof Node) || !event.currentTarget.contains(target)) return;
      if (target instanceof Element && target.closest(interactiveSelector)) return;
      if (window.getSelection()?.toString()) return;
      setFlipped((value) => !value);
    }}>
    <span id={instructionId} style={visuallyHidden}>Press Enter or Space, or use the flip button, to reveal the other side.</span>
    <div style={{ display: 'grid', minHeight: 180, transformStyle: 'preserve-3d', transform: flipped ? 'rotateY(180deg)' : 'rotateY(0deg)', transition: reducedMotion ? 'none' : 'transform 280ms ease' }}>
      <div aria-hidden={flipped} inert={flipped} style={faceStyle}><small style={{ color: muted, fontSize: 11 }}>Front</small><div>{front}</div>{hint ? <small style={{ color: muted, fontSize: 12 }}>Hint: {hint}</small> : null}</div>
      <div aria-hidden={!flipped} inert={!flipped} style={{ ...faceStyle, background: surface, transform: 'rotateY(180deg)' }}><small style={{ color: muted, fontSize: 11 }}>Back</small><div>{back}</div></div>
    </div>
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8, marginTop: 8 }}>
      <span aria-live="polite" style={{ color: muted, fontSize: 12 }}>{flipped ? 'Back' : 'Front'}</span>
      <button type="button" aria-pressed={flipped} onClick={() => setFlipped((value) => !value)} style={buttonStyle}>{flipped ? 'Show front' : 'Show back'}</button>
    </div>
  </section>;
}

const Flashcard = defineComponent({
  name: 'Flashcard', props: z.object({ front: z.any(), back: z.any(), hint: z.string().optional() }),
  description: 'Two-sided card with mouse, touch, and keyboard flipping; respects reduced motion.',
  component: ({ props, renderNode }) => <FlashcardView front={renderNode(props.front)} back={renderNode(props.back)} hint={props.hint} />,
});

const celebrationCSS = `
@keyframes oui-local-confetti-fall {
  0% { transform: translate3d(0, -15vh, 0) rotate(0deg); opacity: 0; }
  10% { opacity: 1; }
  85% { opacity: 1; }
  100% { transform: translate3d(var(--oui-confetti-drift), 110vh, 0) rotate(650deg); opacity: 0; }
}
@media (prefers-reduced-motion: reduce) { .oui-local-confetti-piece { display: none !important; animation: none !important; } }
`;

function CelebrationView({ when }: { when: boolean }) {
  const previous = useRef(false);
  const [burst, setBurst] = useState(0);
  const [visible, setVisible] = useState(false);
  const reducedMotion = useReducedMotion();
  useEffect(() => {
    if (when && !previous.current) { setBurst((value) => value + 1); setVisible(true); }
    previous.current = when;
  }, [when]);
  useEffect(() => {
    if (!visible) return;
    const timer = setTimeout(() => setVisible(false), reducedMotion ? 1400 : 1900);
    return () => clearTimeout(timer);
  }, [visible, burst, reducedMotion]);
  if (!visible || typeof document === 'undefined') return null;
  const colors = ['#3281cd', '#22a785', '#d69429', '#aa69d5', '#d65e8b'];
  return createPortal(<div className="oui-celebration" style={{ position: 'fixed', inset: 0, zIndex: 1500, pointerEvents: 'none', overflow: 'hidden' }}>
    <span role="status" style={visuallyHidden}>Celebration</span>
    {reducedMotion ? <div aria-hidden="true" style={{ position: 'absolute', top: 18, left: '50%', transform: 'translateX(-50%)', fontSize: 32 }}>🎉</div> : <>
      <style>{celebrationCSS}</style>
      {Array.from({ length: 28 }, (_, index) => <span key={`${burst}-${index}`} aria-hidden="true" className="oui-local-confetti-piece" style={{ position: 'absolute', top: 0, left: `${(index * 37) % 101}%`, width: index % 3 === 0 ? 7 : 10, height: index % 2 === 0 ? 12 : 7, background: colors[index % colors.length], borderRadius: index % 3 === 0 ? '50%' : 2, opacity: 0, '--oui-confetti-drift': `${((index * 53) % 181) - 90}px`, animation: `oui-local-confetti-fall 1500ms ${index % 7 * 40}ms ease-in both` } as CSSProperties} />)}
    </>}
  </div>, document.body);
}

const Celebration = defineComponent({
  name: 'Celebration', props: z.object({ when: z.boolean() }),
  description: 'Local, time-bounded confetti on a rising condition; static feedback for reduced motion.',
  component: ({ props }) => <CelebrationView when={props.when} />,
});

export const interactiveComponents: DefinedComponent<any>[] = [Popover, Flashcard, Celebration];
