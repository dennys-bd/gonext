import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/lib/api";
import { updateWidget } from "./actions";

vi.mock("@/lib/api", () => ({
  api: { example: { updateWidget: vi.fn() } },
}));

const request = vi.mocked(api.example.updateWidget);

function fields(): FormData {
  const form = new FormData();
  form.set("count", "1");
  form.set("email", "email value");
  form.set("enabled", "on");
  form.set("note", "note value");
  form.set("password", "password value");
  form.set("price", "2");
  return form;
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("updateWidget", () => {
  it("sends the form and returns the response", async () => {
    request.mockResolvedValue({
      data: { count: 1, enabled: true, name: "name value", tags: [] },
      error: undefined,
      request: new Request("http://localhost"),
      response: new Response(),
    });

    const state = await updateWidget({ slug: ["example"] }, {}, fields());

    expect(request).toHaveBeenCalledWith({ path: { slug: "example" }, body: { count: 1, email: "email value", enabled: true, note: "note value", password: "password value", price: 2, tags: [] } });
    expect(state).toEqual({
      done: true,
      data: { count: 1, enabled: true, name: "name value", tags: [] },
    });
  });

  it("returns the error detail", async () => {
    request.mockResolvedValue({
      data: undefined,
      error: { detail: "rejected" },
      request: new Request("http://localhost"),
      response: new Response(null, { status: 400 }),
    });

    const state = await updateWidget({ slug: ["example"] }, {}, fields());

    expect(state).toEqual({ error: "rejected" });
  });
});
