import { redirect } from "next/navigation";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/lib/api";
import { logout } from "./actions";
import { login } from "./login/actions";

vi.mock("next/navigation", () => ({ redirect: vi.fn() }));
vi.mock("@/lib/api", () => ({
  api: { users: { loginUser: vi.fn(), logoutUser: vi.fn() } },
}));

const loginUser = vi.mocked(api.users.loginUser);
const logoutUser = vi.mocked(api.users.logoutUser);

function credentials(email: string, password: string): FormData {
  const form = new FormData();
  form.set("email", email);
  form.set("password", password);
  return form;
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("login", () => {
  it("sends the form credentials and redirects home on success", async () => {
    loginUser.mockResolvedValue({
      data: {
        id: "u1",
        email: "ada@example.com",
        emailVerified: true,
        role: "user",
        permissions: [],
      },
      error: undefined,
      request: new Request("http://localhost"),
      response: new Response(),
    });

    const state = await login({}, credentials("ada@example.com", "hunter2"));

    expect(loginUser).toHaveBeenCalledWith({
      body: { email: "ada@example.com", password: "hunter2" },
    });
    expect(redirect).toHaveBeenCalledWith("/");
    expect(state).toBeUndefined();
  });

  it("returns the backend's error detail when the credentials are rejected", async () => {
    loginUser.mockResolvedValue({
      data: undefined,
      error: { detail: "invalid credentials", status: 401 },
      request: new Request("http://localhost"),
      response: new Response(null, { status: 401 }),
    });

    const state = await login({}, credentials("ada@example.com", "wrong"));

    expect(state).toEqual({ error: "invalid credentials" });
    expect(redirect).not.toHaveBeenCalled();
  });
});

describe("logout", () => {
  it("revokes the session and redirects to the login page", async () => {
    logoutUser.mockResolvedValue({
      data: undefined,
      error: undefined,
      request: new Request("http://localhost"),
      response: new Response(null, { status: 204 }),
    });

    await logout();

    expect(logoutUser).toHaveBeenCalledTimes(1);
    expect(redirect).toHaveBeenCalledWith("/login");
  });
});
