import { createConnectTransport } from "@connectrpc/connect-web";
import type { Transport } from "@connectrpc/connect";

export interface CreateApiTransportOptions {
  baseUrl?: string;
  storageKey?: string;
}

export function getStoredAccessToken(storageKey: string = "qs_access_token"): string | null {
  if (typeof window === "undefined" || !window.localStorage) return null;
  const token = localStorage.getItem(storageKey);
  if (!token || token === "null" || token === "undefined" || token.trim() === "") return null;
  return token;
}

export function setStoredAccessToken(
  token: string | null,
  storageKey: string = "qs_access_token",
): void {
  if (typeof window === "undefined" || !window.localStorage) return;
  if (token) {
    localStorage.setItem(storageKey, token);
  } else {
    localStorage.removeItem(storageKey);
  }
}

export function getApiBaseUrl(): string {
  if (typeof window === "undefined") {
    return "";
  }

  // Allow explicit override if configured via build/runtime env var
  const envUrl = (import.meta as any).env?.VITE_API_URL;
  if (envUrl) {
    return envUrl;
  }

  const { hostname, protocol } = window.location;

  // Local development / sandbox: separate ports (API server on port 8081)
  if (hostname === "localhost" || hostname === "127.0.0.1") {
    return `${protocol}//${hostname}:8081`;
  }

  // Production / Staging: sub-domains
  if (hostname === "quadsmith.com" || hostname === "www.quadsmith.com") {
    return `${protocol}//api.quadsmith.com`;
  }
  if (hostname.endsWith(".quadsmith.com")) {
    const sub = hostname.replace(/\.?quadsmith\.com$/, "");
    return `${protocol}//${sub}-api.quadsmith.com`;
  }

  // Fallback for custom domains: prefix with "api."
  return `${protocol}//api.${hostname}`;
}

export function createApiTransport(options: CreateApiTransportOptions = {}): Transport {
  const { baseUrl = getApiBaseUrl(), storageKey = "qs_access_token" } = options;
  return createConnectTransport({
    baseUrl,
    fetch: (input, init) => {
      return fetch(input, {
        ...init,
        credentials: "include",
      });
    },
    interceptors: [
      (next) => async (req) => {
        const token = getStoredAccessToken(storageKey);
        if (token) {
          req.header.set("Authorization", `Bearer ${token}`);
        }
        return await next(req);
      },
    ],
  });
}
