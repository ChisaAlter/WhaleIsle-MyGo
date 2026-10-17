// Independently authored resource-backed images. All requests use the parent bridge.
import React, { useContext, useEffect, useMemo, useState, type CSSProperties } from 'react';
import { defineComponent, type DefinedComponent } from '@openuidev/react-lang';
import { z } from 'zod/v4';
import { requestResource } from '../resources';
import { ProgramRevisionContext } from '../capabilities';

const radiusSchema = z.union([z.enum(['sm', 'md', 'lg', 'xl', 'full']), z.number()]);
const fitSchema = z.enum(['cover', 'contain']);
const border = 'var(--oui-border, #d9e0e8)';
const muted = 'var(--oui-muted, #526379)';
const surface = 'var(--oui-surface, #f3f6fa)';
const maxDataLength = 25 * 1024 * 1024;

type ImageStyle = { aspectRatio?: number; radius?: string | number; fit?: 'cover' | 'contain' };
type Photo = { data: string; source: string };
type ImageResponse = { images?: unknown; imageSources?: unknown };
type SearchState = { query: string; status: 'loading' | 'ready' | 'error'; photos: Photo[]; error?: string };
export type PublicImageProps = ImageStyle & { src: string; alt: string; inline?: boolean };
type PublicSource = { kind: 'local'; data: string } | { kind: 'external'; url: string } | { kind: 'invalid'; error: string };
type PublicImageIdentity = { src: string; programRevision: number };
type PublicImageState = { identity: PublicImageIdentity; attempt: number; status: 'loading' | 'decoding' | 'ready' | 'error'; photo?: Photo; error?: string };

function isEncodedBitmap(value: unknown, mediaType: RegExp): value is string {
  if (typeof value !== 'string' || value.length > maxDataLength) return false;
  const prefix = mediaType.exec(value);
  if (!prefix) return false;
  const payload = value.slice(prefix[0].length);
  return payload.length > 0 && payload.length % 4 === 0 && /^[A-Za-z0-9+/]*={0,2}$/.test(payload);
}

function isBitmapData(value: unknown): value is string {
  return isEncodedBitmap(value, /^data:image\/(?:png|jpeg|gif|webp);base64,/i);
}

function isHostBitmapData(value: unknown): value is string {
  // The host validates the actual bytes and image budgets for these formats.
  return isEncodedBitmap(value, /^data:image\/(?:png|jpeg|gif|webp|bmp|x-icon|avif);base64,/i);
}

function publicImageURL(value: string): string | null {
  if (/[\u0000-\u001f\u007f-\u009f\\]/.test(value)) return null;
  const candidate = value.trim();
  const authority = /^https?:\/\/([^/?#]*)/i.exec(candidate);
  if (!authority || authority[1].includes('@')) return null;
  try {
    const url = new URL(candidate);
    if ((url.protocol !== 'http:' && url.protocol !== 'https:') || !url.hostname || url.username || url.password) return null;
    url.hash = '';
    return url.href;
  } catch { return null; }
}

function publicSource(src: string): PublicSource {
  if (isBitmapData(src)) return { kind: 'local', data: src };
  if (/^data:/i.test(src)) return { kind: 'invalid', error: 'Unsupported image data. Only bounded PNG, JPEG, GIF, and WebP base64 bitmaps can be displayed locally.' };
  const url = publicImageURL(src);
  return url ? { kind: 'external', url } : { kind: 'invalid', error: 'Image URLs must be absolute HTTP or HTTPS URLs without credentials or control characters.' };
}

function readPublicPhoto(response: unknown): Photo {
  if (!response || typeof response !== 'object' || Array.isArray(response)) throw new Error('The image service returned an invalid image/source response.');
  const { dataUrl, source, validatedSvg } = response as { dataUrl?: unknown; source?: unknown; validatedSvg?: unknown };
  // SVG is accepted only from the trusted resource response with an explicit
  // host validation marker. User-provided data URLs never take this path.
  const hostValidatedSvg = validatedSvg === true && isEncodedBitmap(dataUrl, /^data:image\/svg\+xml;base64,/i);
  if (!isHostBitmapData(dataUrl) && !hostValidatedSvg) throw new Error('The image service returned an unsupported or oversized image.');
  if (typeof source !== 'string' || !publicImageURL(source)) throw new Error('The image service omitted a valid final HTTP or HTTPS image source.');
  return { data: dataUrl as string, source };
}

function imageStyles(props: ImageStyle): CSSProperties {
  let radius: number | undefined;
  if (typeof props.radius === 'number' && Number.isFinite(props.radius)) radius = Math.min(10000, Math.max(0, props.radius));
  if (typeof props.radius === 'string') {
    const sizes: Record<string, number> = { sm: 4, md: 8, lg: 12, xl: 20, full: 9999 };
    radius = Object.hasOwn(sizes, props.radius) ? sizes[props.radius] : undefined;
  }
  const aspectRatio = typeof props.aspectRatio === 'number' && Number.isFinite(props.aspectRatio) && props.aspectRatio > 0
    ? Math.min(20, Math.max(0.05, props.aspectRatio)) : undefined;
  return { display: 'block', width: '100%', maxWidth: '100%', height: 'auto', aspectRatio, objectFit: props.fit || 'cover', borderRadius: radius, backgroundColor: surface };
}

function ImageStatus({ children, error = false, inline = false }: { children: React.ReactNode; error?: boolean; inline?: boolean }) {
  const Status = inline ? 'span' : 'div';
  return <Status className="oui-image-status" role="status" style={{ display: inline ? 'inline-block' : undefined, minWidth: 0, padding: 16, border: `1px solid ${border}`, borderRadius: 8, backgroundColor: surface, color: error ? 'var(--oui-danger, #b43245)' : muted, fontSize: 13, lineHeight: 1.5, overflowWrap: 'anywhere' }}>{children}</Status>;
}

// Shared by Image and Markdown/Text. Inline output stays phrasing-only and has
// no interactive descendants, including when nested inside a HostLink button.
export function PublicImage({ src, alt, inline = false, ...style }: PublicImageProps) {
  const programRevision = useContext(ProgramRevisionContext);
  const identity = useMemo(() => ({ src, programRevision }), [src, programRevision]);
  const input = useMemo(() => publicSource(src), [src]);
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<PublicImageState>(() => ({ identity, attempt: 0, status: 'loading' }));

  useEffect(() => {
    const controller = new AbortController();
    let live = true;
    const update = (next: Pick<PublicImageState, 'status' | 'photo' | 'error'>) => {
      if (live && !controller.signal.aborted) setState({ identity, attempt, ...next });
    };
    update({ status: 'loading' });
    if (input.kind === 'invalid') update({ status: 'error', error: input.error });
    else if (input.kind === 'local') update({ status: 'decoding', photo: { data: input.data, source: '' } });
    else {
      requestResource<unknown>('/api/image', { src: input.url }, controller.signal)
        .then((response) => {
          if (!live || controller.signal.aborted) return;
          update({ status: 'decoding', photo: readPublicPhoto(response) });
        })
        .catch((error: unknown) => {
          if (!live || controller.signal.aborted) return;
          update({ status: 'error', error: error instanceof Error ? error.message : 'The image service request failed.' });
        });
    }
    return () => { live = false; controller.abort(); };
  }, [identity, input, attempt]);

  // Effects clean up after render: hide the previous image synchronously when
  // its source, trusted program revision, or retry attempt no longer matches.
  const current: PublicImageState = state.identity === identity && state.attempt === attempt
    ? state : { identity, attempt, status: 'loading' };
  const onDecode = (success: boolean) => setState((value) => {
    if (value.identity !== identity || value.attempt !== attempt || (value.status !== 'decoding' && value.status !== 'ready')) return value;
    return success ? { ...value, status: 'ready' } : { ...value, status: 'error', error: 'This image could not be decoded by the viewer. No replacement image was substituted.' };
  });
  const failure = input.kind === 'invalid' ? input.error : current.status === 'error' ? current.error || 'The image service request failed.' : undefined;
  const Frame = inline ? 'span' : 'figure';
  const Caption = inline ? 'span' : 'figcaption';
  return <Frame className="oui-image" style={{ display: inline ? 'inline-block' : undefined, verticalAlign: inline ? 'middle' : undefined, margin: 0, minWidth: 0, maxWidth: '100%' }}>
    {failure ? <ImageStatus error inline={inline}>
      <span>{failure}{alt ? ` Image description: ${alt}` : ''}</span>
      {!inline && input.kind !== 'invalid' ? <button type="button" onClick={() => setAttempt((value) => value + 1)} style={{ ...buttonStyle, display: 'block' }}>Retry image</button> : null}
    </ImageStatus> : <>
      {current.status === 'loading' || current.status === 'decoding' ? <ImageStatus inline={inline}>{current.status === 'decoding' ? 'Decoding image' : 'Loading image'}{alt ? `: ${alt}` : ''}…</ImageStatus> : null}
      {(current.status === 'decoding' || current.status === 'ready') && current.photo ? <img
        src={current.photo.data} alt={alt} style={{ ...imageStyles(style), display: inline ? 'inline-block' : 'block', verticalAlign: inline ? 'middle' : undefined }}
        loading="lazy" decoding="async" draggable={false} onLoad={() => onDecode(true)} onError={() => onDecode(false)}
      /> : null}
    </>}
    {current.photo?.source ? <Caption style={{ display: 'block', marginTop: 6, fontSize: 11, color: muted, lineHeight: 1.5, overflowWrap: 'anywhere' }}>Image source: {current.photo.source}</Caption> : null}
  </Frame>;
}

function DataPhoto({ src, alt, source, ...style }: ImageStyle & { src: string; alt: string; source?: string }) {
  const [failedSource, setFailedSource] = useState<string | null>(null);
  return <figure className="oui-image" style={{ margin: 0, minWidth: 0 }}>
    {failedSource === src
      ? <ImageStatus error>Image could not be decoded: {alt || 'unnamed image'}. No replacement image was substituted.</ImageStatus>
      : <img src={src} alt={alt} style={imageStyles(style)} loading="lazy" decoding="async" draggable={false} onError={() => setFailedSource(src)} />}
    {source ? <figcaption style={{ marginTop: 6, fontSize: 11, color: muted, lineHeight: 1.5, overflowWrap: 'anywhere' }}>
      <details><summary style={{ cursor: 'pointer' }}>Image source</summary><div style={{ marginTop: 4 }}>Source: {source}</div><div>Author and license metadata were not supplied by the image service. External navigation is disabled.</div></details>
    </figcaption> : null}
  </figure>;
}

function readPhotos(response: ImageResponse): Photo[] {
  if (!response || typeof response !== 'object' || !Array.isArray(response.images) || !Array.isArray(response.imageSources)) {
    throw new Error('The image service returned an invalid image/source response.');
  }
  if (response.images.length !== response.imageSources.length || response.images.length > 4) {
    throw new Error('The image service returned mismatched images and source metadata.');
  }
  const images = response.images;
  const sources = response.imageSources;
  return images.map((value, index) => {
    const source = sources[index];
    if (!isBitmapData(value)) throw new Error('The image service returned an unsupported or oversized bitmap.');
    if (typeof source !== 'string' || source.trim() === '') throw new Error('The image service omitted the original image source.');
    return { data: value, source };
  });
}

function SearchPhoto({ query, ...style }: ImageStyle & { query: string }) {
  const normalizedQuery = query.trim().replace(/\s+/g, ' ');
  const queryLength = Array.from(normalizedQuery).length;
  const validQuery = queryLength >= 2 && queryLength <= 200;
  const [state, setState] = useState<SearchState>({ query: normalizedQuery, status: 'loading', photos: [] });
  const [selection, setSelection] = useState(0);
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    let live = true;
    setSelection(0);
    if (!validQuery) {
      setState({ query: normalizedQuery, status: 'error', photos: [], error: 'Image search requires a query between 2 and 200 characters.' });
      return () => { live = false; controller.abort(); };
    }
    setState({ query: normalizedQuery, status: 'loading', photos: [] });
    requestResource<ImageResponse>('/api/image-search', { q: normalizedQuery }, controller.signal)
      .then((response) => {
        if (!live) return;
        setState({ query: normalizedQuery, status: 'ready', photos: readPhotos(response) });
      })
      .catch((error: unknown) => {
        if (!live || controller.signal.aborted) return;
        const message = error instanceof Error ? error.message : 'The image service request failed.';
        setState({ query: normalizedQuery, status: 'error', photos: [], error: message });
      });
    return () => { live = false; controller.abort(); };
  }, [normalizedQuery, validQuery, revision]);

  if (!validQuery) return <ImageStatus error>Image search requires a query between 2 and 200 characters.</ImageStatus>;
  if (state.query !== normalizedQuery || state.status === 'loading') return <ImageStatus>Searching for images of “{normalizedQuery}”…</ImageStatus>;
  if (state.status === 'error') return <ImageStatus error><div>Image search failed: {state.error}</div><button type="button" onClick={() => setRevision((value) => value + 1)} style={buttonStyle}>Retry image search</button></ImageStatus>;
  if (state.photos.length === 0) return <ImageStatus>No images were returned for “{normalizedQuery}”.</ImageStatus>;
  const index = Math.min(selection, state.photos.length - 1);
  const photo = state.photos[index];
  return <div className="oui-search-image" style={{ minWidth: 0 }}>
    <DataPhoto src={photo.data} source={photo.source} alt={`Image result for ${normalizedQuery}`} {...style} />
    {state.photos.length > 1 ? <div role="group" aria-label="Image search results" style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8, color: muted, fontSize: 12 }}>
      <button type="button" aria-label="Previous image" disabled={index === 0} onClick={() => setSelection((value) => Math.max(0, value - 1))} style={buttonStyle}>Previous</button>
      <span aria-live="polite">{index + 1} of {state.photos.length}</span>
      <button type="button" aria-label="Next image" disabled={index === state.photos.length - 1} onClick={() => setSelection((value) => Math.min(state.photos.length - 1, value + 1))} style={buttonStyle}>Next</button>
    </div> : null}
  </div>;
}

const buttonStyle: CSSProperties = { border: `1px solid ${border}`, background: 'var(--oui-panel, #fff)', color: 'var(--oui-ink, #172337)', padding: '5px 9px', borderRadius: 6, font: 'inherit', marginTop: 4, cursor: 'pointer' };

const Image = defineComponent({
  name: 'Image',
  props: z.object({ src: z.string(), alt: z.string(), aspectRatio: z.number().optional(), radius: radiusSchema.optional(), fit: fitSchema.optional() }),
  description: 'A local bitmap or public HTTP(S) image loaded through the parent image service, with visible source and loading/error states. No replacement-image fallback.',
  component: ({ props }) => <PublicImage {...props} />,
});

const SearchImage = defineComponent({
  name: 'SearchImage',
  props: z.object({ query: z.string(), aspectRatio: z.number().optional(), radius: radiusSchema.optional(), fit: fitSchema.optional() }),
  description: 'Searches the parent image service for real bitmap images and retains their original source URLs.',
  component: ({ props }) => <SearchPhoto {...props} />,
});

export const imageComponents: DefinedComponent<any>[] = [Image, SearchImage];
