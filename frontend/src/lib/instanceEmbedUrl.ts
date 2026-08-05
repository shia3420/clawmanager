import { APP_BASE } from "./appBase";

export function resolveInstanceEmbedUrl(url: string | null): string | null {
  if (!url) {
    return null;
  }
  if (/^https?:\/\//i.test(url)) {
    return url;
  }

  const explicitOrigin = import.meta.env.VITE_BACKEND_ORIGIN as
    | string
    | undefined;
  if (explicitOrigin) {
    return new URL(url, explicitOrigin).toString();
  }

  if (url.startsWith("/api/")) {
    return `${APP_BASE}${url.replace(/^\/+/, "")}`;
  }

  return url;
}
