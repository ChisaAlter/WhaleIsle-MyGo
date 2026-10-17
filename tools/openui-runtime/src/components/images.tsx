// Independently authored resource-backed images. All requests use the parent bridge.
import React, { useEffect, useState, type CSSProperties } from 'react';
import { defineComponent, type DefinedComponent } from '@openuidev/react-lang';
import { z } from 'zod/v4';
import { requestResource } from '../resources';

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

function isBitmapData(value: unknown): value is string {
  if (typeof value !== 'string' || value.length > maxDataLength) return false;
  const prefix = /^data:image\/(?:png|jpeg|gif|webp);base64,/i.exec(value);
  if (!prefix) return false;
  const payload = value.slice(prefix[0].length);
  return payload.length > 0 && payload.length % 4 === 0 && /^[A-Za-z0-9+/]*={0,2}$/.test(payload);
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

function ImageStatus({ children, error = false }: { children: React.ReactNode; error?: boolean }) {
  return <div className="oui-image-status" role="status" style={{ minWidth: 0, padding: 16, border: `1px solid ${border}`, borderRadius: 8, backgroundColor: surface, color: error ? 'var(--oui-danger, #b43245)' : muted, fontSize: 13, lineHeight: 1.5, overflowWrap: 'anywhere' }}>{children}</div>;
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
  description: 'A local bitmap data image. External URLs and replacement-image fallback are unsupported.',
  component: ({ props }) => {
    if (!isBitmapData(props.src)) return <ImageStatus error>{props.src.startsWith('data:') ? 'Unsupported image data. Only bounded PNG, JPEG, GIF, and WebP base64 bitmaps can be displayed.' : 'External image URLs cannot be opened in this isolated viewer. Use SearchImage for the parent image service.'}{props.alt ? <div>Image description: {props.alt}</div> : null}</ImageStatus>;
    return <DataPhoto {...props} />;
  },
});

const SearchImage = defineComponent({
  name: 'SearchImage',
  props: z.object({ query: z.string(), aspectRatio: z.number().optional(), radius: radiusSchema.optional(), fit: fitSchema.optional() }),
  description: 'Searches the parent image service for real bitmap images and retains their original source URLs.',
  component: ({ props }) => <SearchPhoto {...props} />,
});

export const imageComponents: DefinedComponent<any>[] = [Image, SearchImage];
