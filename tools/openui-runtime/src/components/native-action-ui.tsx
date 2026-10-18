// Independently authored controls. Only the trusted Host may perform these actions.
import React, { useContext, useId, useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { useIsStreaming } from '@openuidev/react-lang';
import { HostActionsContext } from '../capabilities';
import { prepareNativeActions, performNativeAction, type GestureEvent, type NativeAction } from '../native-actions';

type ActionState = { phase: 'idle' | 'pending' | 'success' | 'error'; message: string };
const idleState: ActionState = { phase: 'idle', message: '' };
const muted = 'var(--oui-muted, #526379)';

function HostActionControl({ action, label }: { action: NativeAction; label: ReactNode }) {
  const supported = useContext(HostActionsContext);
  const streaming = useIsStreaming();
  const statusId = useId();
  const [state, setState] = useState<ActionState>(idleState);
  const lifetime = useRef<{ mounted: boolean; request: object | null }>({ mounted: false, request: null });
  const isCopy = action.type === 'clipboard';

  useLayoutEffect(() => {
    const current = lifetime.current;
    current.mounted = true;
    return () => {
      // A replacement payload has its own keyed lifetime. The previous Host
      // operation may finish, but must never update the replacement or unmounted UI.
      current.mounted = false;
      current.request = null;
    };
  }, []);

  function invoke(event: GestureEvent) {
    const current = lifetime.current;
    // The ref closes the same-render double-click gap before React can disable
    // the button. Preparing the permit stays inside the actual click callback.
    if (!current.mounted || current.request || streaming || !supported) return;
    const request = {};
    current.request = request;
    const finish = (next: ActionState) => {
      if (!current.mounted || current.request !== request) return;
      current.request = null;
      setState(next);
    };
    const fail = (error: unknown) => finish({
      phase: 'error',
      message: error instanceof Error && error.message ? error.message : 'The Host could not complete this action.',
    });
    try {
      // Preserve the original clipboard text and URL. Payload and trusted-event
      // validation belong to the shared bridge, not DOM text or browser APIs.
      const permit = prepareNativeActions(event, [action]);
      setState({ phase: 'pending', message: isCopy ? 'Copying…' : 'Sending to browser…' });
      void performNativeAction(permit).then(
        () => finish({ phase: 'success', message: isCopy ? 'Copied' : 'Sent to browser' }),
        fail,
      );
    } catch (error: unknown) {
      fail(error);
    }
  }

  const pending = state.phase === 'pending';
  const disabled = pending || streaming || !supported;
  const unavailable = !supported
    ? (isCopy ? 'Clipboard access is unavailable in this Host.' : 'Opening links is unavailable in this Host.')
    : streaming ? 'Available when the response finishes.' : '';
  const message = state.message || unavailable;
  const buttonStyle: CSSProperties = isCopy
    ? { border: '1px solid var(--oui-border, #d9e0e8)', borderRadius: 5, padding: '3px 8px', background: 'transparent', color: 'var(--oui-ink, #172337)', font: 'inherit' }
    : { border: 0, padding: 0, background: 'transparent', color: '#245fba', textDecoration: 'underline', font: 'inherit', textAlign: 'inherit', overflowWrap: 'anywhere' };
  return <span style={{ display: 'inline-flex', flexWrap: 'wrap', alignItems: 'baseline', gap: 6, maxWidth: '100%' }}>
    <button
      type="button"
      role={isCopy ? undefined : 'link'}
      disabled={disabled}
      aria-disabled={disabled}
      aria-busy={pending}
      aria-describedby={message ? statusId : undefined}
      aria-label={isCopy ? 'Copy original code' : undefined}
      title={isCopy ? unavailable || undefined : action.type === 'open-url' ? action.url : undefined}
      onClick={invoke}
      style={{ ...buttonStyle, cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.6 : 1 }}
    >{label}{!isCopy && <span style={{ position: 'absolute', width: 1, height: 1, padding: 0, margin: -1, overflow: 'hidden', clipPath: 'inset(50%)', whiteSpace: 'nowrap', border: 0 }}> (opens in external browser)</span>}</button>
    <span id={statusId} role={state.phase === 'error' ? 'alert' : 'status'} aria-live={state.phase === 'error' ? 'assertive' : 'polite'} aria-atomic="true" style={{ color: state.phase === 'error' ? 'var(--oui-danger, #b43245)' : muted, fontSize: 12, overflowWrap: 'anywhere' }}>{message}</span>
  </span>;
}

export function CodeCopyButton({ code, language }: { code: string; language?: string }) {
  // The complete source is the action payload, even when highlighting falls back.
  // Keys reset feedback and synchronously retire old requests when content changes.
  return <HostActionControl key={JSON.stringify([code, language])} action={{ type: 'clipboard', text: code }} label="Copy" />;
}

export function HostLink({ text, url, children }: { text?: string; url: string; children?: ReactNode }) {
  // No href, window navigation, prefetch, or automatic action on mount/update.
  // The payload owns the request lifetime. React children retain their formatting
  // and accessible text; never stringify their elements, owners or cyclic fibers.
  const label = React.Children.count(children) > 0 ? children : text || url;
  return <HostActionControl key={url} action={{ type: 'open-url', url }} label={label} />;
}
