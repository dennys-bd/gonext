import { cookies } from "next/headers";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createClientConfig } from "./api-client";

vi.mock("next/headers", () => ({ cookies: vi.fn() }));

const cookieStore = { get: vi.fn(), set: vi.fn() };
const fetchMock = vi.fn();

function relay(response: Response) {
  fetchMock.mockResolvedValueOnce(response);
  const config = createClientConfig({});
  if (!config.fetch) throw new Error("createClientConfig did not set fetch");
  return config.fetch(new Request("http://localhost:8080/users/me"));
}

function sentRequest(): Request {
  return fetchMock.mock.calls[0][0];
}

describe("createClientConfig", () => {
  beforeEach(() => {
    vi.mocked(cookies).mockResolvedValue(
      cookieStore as unknown as Awaited<ReturnType<typeof cookies>>,
    );
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
  });

  it("uses API_URL as the base URL when set", () => {
    vi.stubEnv("API_URL", "http://api.internal:9000");
    expect(createClientConfig({}).baseUrl).toBe("http://api.internal:9000");
  });

  it("defaults the base URL to the local backend", () => {
    vi.stubEnv("API_URL", "");
    expect(createClientConfig({}).baseUrl).toBe("http://localhost:8080");
  });

  it("relays the incoming session cookie to the backend", async () => {
    cookieStore.get.mockReturnValue({ name: "session", value: "abc" });

    await relay(new Response(null, { status: 200 }));

    expect(sentRequest().headers.get("cookie")).toBe("session=abc");
  });

  it("sends no Cookie header when there is no session", async () => {
    cookieStore.get.mockReturnValue(undefined);

    await relay(new Response(null, { status: 200 }));

    expect(sentRequest().headers.get("cookie")).toBeNull();
  });

  it("applies a Set-Cookie from the backend with its attributes preserved", async () => {
    const expires = "Wed, 21 Oct 2026 07:28:00 GMT";
    const response = new Response(null, {
      status: 200,
      headers: {
        "set-cookie": `session=xyz; Path=/; Expires=${expires}; HttpOnly; Secure; SameSite=Lax`,
      },
    });

    await relay(response);

    expect(cookieStore.set).toHaveBeenCalledTimes(1);
    expect(cookieStore.set).toHaveBeenCalledWith({
      name: "session",
      value: "xyz",
      path: "/",
      expires: new Date(expires),
      httpOnly: true,
      secure: true,
      sameSite: "lax",
    });
  });

  it("applies a clearing Set-Cookie (logout in a relaxed env)", async () => {
    const response = new Response(null, {
      status: 204,
      headers: {
        "set-cookie": "session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax",
      },
    });

    await relay(response);

    expect(cookieStore.set).toHaveBeenCalledWith({
      name: "session",
      value: "",
      path: "/",
      maxAge: 0,
      httpOnly: true,
      secure: false,
      sameSite: "lax",
    });
  });

  it("ignores Set-Cookie headers for anything but the session cookie", async () => {
    const headers = new Headers();
    headers.append("set-cookie", "tracking=1; Path=/");
    headers.append("set-cookie", "session=xyz; Path=/; HttpOnly");

    await relay(new Response(null, { status: 200, headers }));

    expect(cookieStore.set).toHaveBeenCalledTimes(1);
    expect(cookieStore.set).toHaveBeenCalledWith(
      expect.objectContaining({ name: "session", value: "xyz" }),
    );
  });

  it("never touches the cookie store when the backend sets no cookie", async () => {
    await relay(new Response(null, { status: 200 }));

    expect(cookieStore.set).not.toHaveBeenCalled();
  });
});
