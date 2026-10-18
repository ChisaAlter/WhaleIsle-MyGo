// Independently authored static HTML script preparation. parse5 builds only an
// inert syntax tree: no caller HTML or JavaScript enters the viewer document.
import { parse, type DefaultTreeAdapterMap } from "parse5";
import { requestResource } from "./resources";

export type HtmlScriptSource = { source: string; integrityVerified: boolean };
export type PreparedHtmlScripts = { html: string; sources: HtmlScriptSource[] };
type HtmlElement = DefaultTreeAdapterMap["element"];
type Range = { startOffset: number; endOffset: number };
type Replacement = Range & { text: string };
type Script = { src: string; integrity?: string; srcRange: Range; remove: Range[] };
type ScriptResponse = HtmlScriptSource & { dataUrl: string };

const HTML_NAMESPACE = "http://www.w3.org/1999/xhtml";
const DATA_PREFIX = "data:text/javascript;base64,";
// HTML uses a JavaScript MIME *essence match*, not MIME parsing: parameters do
// not qualify. Legacy language="..." becomes text/... only without type.
const JAVASCRIPT_TYPES = new Set([
  "application/ecmascript", "application/javascript", "application/x-ecmascript", "application/x-javascript",
  "text/ecmascript", "text/javascript", "text/javascript1.0", "text/javascript1.1", "text/javascript1.2",
  "text/javascript1.3", "text/javascript1.4", "text/javascript1.5", "text/jscript", "text/livescript",
  "text/x-ecmascript", "text/x-javascript",
]);
const asciiLower = (value: string) => value.replace(/[A-Z]/g, letter => letter.toLowerCase());
const asciiTrim = (value: string) => value.replace(/^[\t\n\f\r ]+|[\t\n\f\r ]+$/g, "");

function cancelled(signal: AbortSignal) {
  if (signal.aborted) throw new DOMException("HTML script preparation cancelled.", "AbortError");
}

function attribute(element: HtmlElement, name: string): string | undefined {
  return element.attrs.find(attr => !attr.namespace && attr.name === name)?.value;
}

function scriptType(element: HtmlElement): "classic" | "module" | "other" {
  const type = attribute(element, "type");
  const language = attribute(element, "language");
  const effective = type === "" || (type === undefined && !language)
    ? "text/javascript" : type ?? `text/${language}`;
  if (JAVASCRIPT_TYPES.has(asciiLower(effective)) ||
      (type !== undefined && JAVASCRIPT_TYPES.has(asciiLower(asciiTrim(type))))) return "classic";
  // Whitespace stripping is specified for MIME matches only, not module.
  return asciiLower(effective) === "module" ? "module" : "other";
}

function checkedUrl(value: string, finalSource = false): string {
  // Do not let URL's recovery rules strip controls, infer a scheme, normalize
  // backslashes or silently repair a missing authority. Host checks the actual
  // public-network destination and every redirect again before fetching.
  if (!/^https?:\/\/[^/?#]/i.test(value) || /[\s\u0000-\u001f\u007f-\u009f\\]/.test(value)) {
    throw new Error("HtmlView external scripts require an absolute HTTP(S) URL without whitespace, controls or backslashes.");
  }
  let url: URL;
  try { url = new URL(value); }
  catch { throw new Error("HtmlView received an invalid external script URL."); }
  const authority = value.slice(value.indexOf("://") + 3).split(/[/?#]/, 1)[0];
  if ((url.protocol !== "http:" && url.protocol !== "https:") || !url.hostname ||
      url.username || url.password || authority.includes("@") || (finalSource && value.includes("#"))) {
    throw new Error("HtmlView external script URLs cannot contain credentials, and final source URLs cannot contain fragments.");
  }
  return value;
}

function checkedRange(range: Range | undefined, startTag: Range, htmlLength: number): Range {
  if (!range || !Number.isSafeInteger(range.startOffset) || !Number.isSafeInteger(range.endOffset) ||
      range.startOffset < startTag.startOffset || range.endOffset <= range.startOffset ||
      range.endOffset > startTag.endOffset || range.endOffset > htmlLength) {
    throw new Error("HtmlView could not safely locate an external script attribute.");
  }
  return { startOffset: range.startOffset, endOffset: range.endOffset };
}

function collectScripts(html: string): Script[] {
  const duplicateOffsets: number[] = [];
  const document = parse(html, {
    scriptingEnabled: true,
    sourceCodeLocationInfo: true,
    onParseError(error) {
      if (error.code === "duplicate-attribute") duplicateOffsets.push(error.startOffset);
    },
  });
  const scripts: Script[] = [];
  const stack: DefaultTreeAdapterMap["node"][] = [document];
  while (stack.length) {
    const node = stack.pop()!;
    if ("tagName" in node) {
      // Template contents belong to a separate inert document fragment. Never
      // walk that fragment, even if it contains modules or external scripts.
      if (node.namespaceURI === HTML_NAMESPACE && node.tagName === "template") continue;
      if (node.tagName === "script") {
        if (node.namespaceURI !== HTML_NAMESPACE) {
          if (node.attrs.some(attr => attr.name === "src" || attr.name === "href" || attr.name === "xlink:href")) {
            throw new Error("HtmlView external scripts in SVG or other non-HTML namespaces are unsupported.");
          }
        } else {
          const location = node.sourceCodeLocation;
          // EOF in an unclosed HTML script marks it already-started in the
          // browser; it is not an active static script to request or rewrite.
          if (!location?.endTag) continue;
          const type = scriptType(node);
          if (type === "module") throw new Error("HtmlView module scripts are unsupported, including inline modules.");
          const src = attribute(node, "src");
          if (type === "classic" && src !== undefined) {
            const startTag = location.startTag;
            if (!startTag || startTag.startOffset < 0 || startTag.endOffset > html.length) {
              throw new Error("HtmlView could not safely locate an external script start tag.");
            }
            // Removing an effective SRI/CORS attribute must not expose a
            // duplicate ignored by the original HTML parser. Fail only this
            // candidate, rather than rejecting malformed unrelated markup.
            if (duplicateOffsets.some(offset => offset >= startTag.startOffset && offset < startTag.endOffset)) {
              throw new Error("HtmlView cannot prepare an external script with duplicate attributes.");
            }
            const crossorigin = attribute(node, "crossorigin");
            if (crossorigin !== undefined && crossorigin !== "" && asciiLower(crossorigin) !== "anonymous") {
              throw new Error("HtmlView external scripts support only omitted, empty or anonymous crossorigin; credentials are unavailable.");
            }
            const integrity = attribute(node, "integrity");
            const srcRange = checkedRange(location.attrs?.src, startTag, html.length);
            const remove: Range[] = [];
            if (integrity !== undefined) remove.push(checkedRange(location.attrs?.integrity, startTag, html.length));
            if (crossorigin !== undefined) remove.push(checkedRange(location.attrs?.crossorigin, startTag, html.length));
            scripts.push({ src: checkedUrl(src), integrity, srcRange, remove });
          }
        }
      }
    }
    if ("childNodes" in node) {
      for (let index = node.childNodes.length - 1; index >= 0; index--) stack.push(node.childNodes[index]);
    }
  }
  // Tree correction can move nodes. The original source order is the order
  // relevant to replacing parser-inserted scripts and displaying provenance.
  return scripts.sort((left, right) => left.srcRange.startOffset - right.srcRange.startOffset);
}

function checkedResponse(value: unknown, integrity: string | undefined): ScriptResponse {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("HtmlView received an invalid script resource response.");
  const response = value as Record<string, unknown>;
  if (typeof response.dataUrl !== "string" || !response.dataUrl.startsWith(DATA_PREFIX) ||
      typeof response.source !== "string" || typeof response.integrityVerified !== "boolean") {
    throw new Error("HtmlView received an invalid script resource response.");
  }
  const base64 = response.dataUrl.slice(DATA_PREFIX.length);
  const paddingAt = base64.indexOf("=");
  const encoded = paddingAt < 0 ? base64 : base64.slice(0, paddingAt);
  const padding = paddingAt < 0 ? "" : base64.slice(paddingAt);
  if (base64.length % 4 !== 0 || /[^A-Za-z0-9+/]/.test(encoded) || (padding !== "" && padding !== "=" && padding !== "==")) {
    throw new Error("HtmlView received invalid base64 script data.");
  }
  if (integrity !== undefined && asciiTrim(integrity) !== "" && response.integrityVerified !== true) {
    throw new Error("The Host did not verify the external script's requested integrity.");
  }
  return { dataUrl: response.dataUrl, source: checkedUrl(response.source, true), integrityVerified: response.integrityVerified };
}

/** Fetch every eligible static classic script before the caller creates its
 * iframe. No script text is decoded, combined, evaluated or injected here.
 * The resulting currentScript.src is the verified data URL, not the CDN URL.
 */
export async function prepareHtmlScripts(html: string, signal: AbortSignal): Promise<PreparedHtmlScripts> {
  cancelled(signal);
  const scripts = collectScripts(html);
  cancelled(signal);
  const responses = await Promise.all(scripts.map(async script => {
    const query: Record<string, string> = { src: script.src };
    if (script.integrity !== undefined) query.integrity = script.integrity;
    // requestResource owns the existing four-active-request queue and Host
    // response authentication. There is no additional script-count limit.
    const value = await requestResource("/api/html-script", query, signal);
    cancelled(signal);
    return checkedResponse(value, script.integrity);
  }));
  cancelled(signal);
  const replacements: Replacement[] = [];
  scripts.forEach((script, index) => {
    // Strict base64 validation above means this quoted attribute needs no
    // HTML escaping and can never contain markup or a closing script tag.
    replacements.push({ ...script.srcRange, text: `src="${responses[index].dataUrl}"` });
    // Host already checked SRI on the original bytes. Retaining SRI/CORS here
    // would request a second browser check against the opaque data URL.
    script.remove.forEach(range => replacements.push({ ...range, text: "" }));
  });
  replacements.sort((left, right) => left.startOffset - right.startOffset);
  const chunks: string[] = [];
  let offset = 0;
  for (const replacement of replacements) {
    if (replacement.startOffset < offset) throw new Error("HtmlView encountered overlapping script attribute locations.");
    chunks.push(html.slice(offset, replacement.startOffset), replacement.text);
    offset = replacement.endOffset;
  }
  chunks.push(html.slice(offset));
  cancelled(signal);
  return {
    html: chunks.join(""),
    sources: responses.map(({ source, integrityVerified }) => ({ source, integrityVerified })),
  };
}
