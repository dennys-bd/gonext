// Runtime configuration for the generated client in lib/api/ (see
// openapi-ts.config.ts, `runtimeConfigPath`). It is the only hand-written
// piece of the client, and it is server-only by construction: importing
// `next/headers` makes any client-component import a build error.
//
// Every call goes through Next's server, so the browser never talks to Go
// directly. This module relays identity across that hop:
//   - outbound: the browser's session cookie is copied onto the request to Go;
//   - inbound:  any Set-Cookie from Go is applied to the Next response.
// Cookie attributes are copied as Go sent them, never re-derived, so cookie
// policy stays owned by backend/users/internal/presentation/cookie.go.
import { cookies } from "next/headers";
import type { CreateClientConfig } from "./api/client";

// Mirrors auth.DefaultCookieName on the Go side.
export const SESSION_COOKIE = "session";

// Deliberately not NEXT_PUBLIC_: the backend address must not reach the
// browser, and the relay only runs on the server.
const DEFAULT_API_URL = "http://localhost:8080";

const SAME_SITE_VALUES = ["lax", "strict", "none"] as const;
type SameSite = (typeof SAME_SITE_VALUES)[number];

function toSameSite(value: string): SameSite | undefined {
  const lowered = value.toLowerCase();
  return SAME_SITE_VALUES.find((v) => v === lowered);
}

// Attributes cookie.go emits, in the shape cookies().set() accepts.
export type SessionCookie = {
  name: string;
  value: string;
  path?: string;
  expires?: Date;
  maxAge?: number;
  httpOnly: boolean;
  secure: boolean;
  sameSite?: SameSite;
};

function splitPair(pair: string): [string, string] {
  const at = pair.indexOf("=");
  return at === -1 ? [pair, ""] : [pair.slice(0, at), pair.slice(at + 1)];
}

// Handles exactly the attributes cookie.go emits; anything else (Domain,
// Partitioned, …) is dropped here, so extend this switch when Go's cookie
// policy grows.
function withAttribute(cookie: SessionCookie, attr: string): SessionCookie {
  const [key, value] = splitPair(attr);
  switch (key.toLowerCase()) {
    case "path":
      return { ...cookie, path: value };
    case "expires":
      return { ...cookie, expires: new Date(value) };
    case "max-age":
      return { ...cookie, maxAge: Number(value) };
    case "httponly":
      return { ...cookie, httpOnly: true };
    case "secure":
      return { ...cookie, secure: true };
    case "samesite":
      return { ...cookie, sameSite: toSameSite(value) };
    default:
      return cookie;
  }
}

export function parseSetCookie(header: string): SessionCookie {
  const [pair, ...attributes] = header.split(";").map((s) => s.trim());
  const [name, value] = splitPair(pair);
  return attributes.reduce(withAttribute, {
    name,
    value,
    httpOnly: false,
    secure: false,
  });
}

async function withSessionCookie(request: Request): Promise<Request> {
  const session = (await cookies()).get(SESSION_COOKIE);
  if (!session) return request;
  const headers = new Headers(request.headers);
  headers.set("cookie", `${SESSION_COOKIE}=${session.value}`);
  return new Request(request, { headers });
}

// cookies().set() is only legal inside a server action or route handler;
// Next throws if a server component sets a cookie during render. Only
// login-user and logout-user emit Set-Cookie and both run from actions, so
// this write path is never reached from a page. Any future endpoint that
// sets a cookie must likewise be called from an action.
async function applySetCookies(response: Response): Promise<void> {
  const headers = response.headers.getSetCookie();
  if (headers.length === 0) return;
  const store = await cookies();
  for (const header of headers) {
    const cookie = parseSetCookie(header);
    // The relay's trust boundary: only the session cookie crosses to the
    // browser, whatever else the backend may set.
    if (cookie.name !== SESSION_COOKIE) continue;
    store.set(cookie);
  }
}

const relayFetch: typeof fetch = async (input, init) => {
  const request = await withSessionCookie(new Request(input, init));
  const response = await fetch(request);
  await applySetCookies(response);
  return response;
};

export const createClientConfig: CreateClientConfig = (config) => ({
  ...config,
  baseUrl: process.env.API_URL || DEFAULT_API_URL,
  fetch: relayFetch,
});
