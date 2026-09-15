import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/lib/api";
import { logoutUser } from "./actions";

vi.mock("@/lib/api", () => ({
  api: { users: { logoutUser: vi.fn() } },
}));

const request = vi.mocked(api.users.logoutUser);

function fields(): FormData {
  const form = new FormData();
  return form;
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("logoutUser", () => {
  it("sends the form and returns the response", async () => {
    request.mockResolvedValue({
      data: undefined,
      error: undefined,
      request: new Request("http://localhost"),
      response: new Response(),
    });

    const state = await logoutUser({}, fields());

    expect(request).toHaveBeenCalledWith();
    expect(state).toEqual({
      done: true,
    });
  });

  it("returns the error detail", async () => {
    request.mockResolvedValue({
      data: undefined,
      error: { detail: "rejected" },
      request: new Request("http://localhost"),
      response: new Response(null, { status: 400 }),
    });

    const state = await logoutUser({}, fields());

    expect(state).toEqual({ error: "rejected" });
  });
});
