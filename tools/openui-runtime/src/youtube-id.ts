// Independently authored input validation. This module never loads a URL or media.
export type YouTubeVideoIdResult =
  | { videoId: string; error?: never }
  | { videoId?: never; error: string };

const VIDEO_ID = /^[A-Za-z0-9_-]{11}$/;
const MAX_INPUT_LENGTH = 2048;
const YOUTUBE_HOSTS = new Set(["youtube.com", "www.youtube.com", "m.youtube.com"]);
const PRIVATE_HOSTS = new Set(["youtube-nocookie.com", "www.youtube-nocookie.com"]);
const SHORT_HOSTS = new Set(["youtu.be", "www.youtu.be"]);

/** Extract only a video ID; URL options are discarded and existence is not checked. */
export function parseYouTubeVideoId(video: string): YouTubeVideoIdResult {
  if (typeof video !== "string" || video.length === 0 || video.length > MAX_INPUT_LENGTH) {
    return { error: "YouTube video must be a nonempty string of at most 2048 characters." };
  }
  if (video.length === 11 && VIDEO_ID.test(video)) return { videoId: video };

  // Reject syntax that URL parsers may silently trim, normalize, or reinterpret.
  if (/[\s\u0000-\u001f\u007f-\u009f\\]/u.test(video) || /%(?:0[0-9a-f]|1[0-9a-f]|7f)|%c2%(?:8[0-9a-f]|9[0-9a-f])/i.test(video)) {
    return { error: "YouTube URLs cannot contain whitespace, control characters, or backslashes." };
  }
  const parts = /^(https?):\/\/([^/?#]+)(\/[^?#]*)?(\?[^#]*)?(#.*)?$/i.exec(video);
  if (!parts) return { error: "Use an 11-character video ID or an explicit HTTP(S) YouTube URL." };

  const scheme = parts[1].toLowerCase();
  const authority = parts[2];
  if (authority.includes("@")) return { error: "YouTube URLs cannot contain credentials." };
  const hostAndPort = /^([a-z0-9.-]+)(?::([0-9]+))?$/i.exec(authority);
  if (!hostAndPort) return { error: "YouTube URL has an invalid hostname or port." };
  const hostname = hostAndPort[1].toLowerCase();
  const port = hostAndPort[2];
  if (port !== undefined && port !== (scheme === "https" ? "443" : "80")) {
    return { error: "YouTube URLs must use the default HTTP(S) port." };
  }
  if (!YOUTUBE_HOSTS.has(hostname) && !PRIVATE_HOSTS.has(hostname) && !SHORT_HOSTS.has(hostname)) {
    return { error: "YouTube URL hostname is not supported." };
  }

  // Inspect the original path, before URL normalization could hide dot segments.
  const path = parts[3] ?? "/";
  let videoId: string | undefined;
  if (SHORT_HOSTS.has(hostname)) {
    videoId = /^\/([A-Za-z0-9_-]{11})\/?$/.exec(path)?.[1];
  } else if (YOUTUBE_HOSTS.has(hostname) && path === "/watch") {
    const candidates = new URLSearchParams(parts[4] ?? "").getAll("v");
    if (candidates.length !== 1) {
      return { error: "YouTube watch URLs must contain exactly one video ID parameter." };
    }
    videoId = candidates[0];
  } else {
    const route = /^\/(embed|shorts|live)\/([A-Za-z0-9_-]{11})\/?$/.exec(path);
    if (route && (!PRIVATE_HOSTS.has(hostname) || route[1] === "embed")) {
      // These embed routes select a playlist/channel rather than one fixed video.
      if (route[1] === "embed" && (route[2] === "videoseries" || route[2] === "live_stream")) {
        return { error: "YouTube playlist and channel embed targets are not supported." };
      }
      videoId = route[2];
    }
  }

  if (videoId === undefined || videoId.length !== 11 || !VIDEO_ID.test(videoId)) {
    return { error: "YouTube URL must identify one video with an 11-character video ID." };
  }
  return { videoId };
}
