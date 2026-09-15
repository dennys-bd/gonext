import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/lib/api";
import { createStub } from "./actions";

vi.mock("@/lib/api", () => ({
  api: { example: { createStub: vi.fn() } },
}));

const request = vi.mocked(api.example.createStub);

function fields(): FormData {
  const form = new FormData();
  form.set("name", "name value");
  return form;
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("createStub", () => {
  it("sends the form and returns the response", async () => {
    request.mockResolvedValue({
      data: { createdAt: "createdAt value", id: "id value", name: "name value" },
      error: undefined,
      request: new Request("http://localhost"),
      response: new Response(),
    });

    const state = await createStub({}, fields());

    expect(request).toHaveBeenCalledWith({ body: { name: "name value" } });
    expect(state).toEqual({
      done: true,
      data: { createdAt: "createdAt value", id: "id value", name: "name value" },
    });
  });

  it("returns the error detail", async () => {
    request.mockResolvedValue({
      data: undefined,
      error: { detail: "rejected" },
      request: new Request("http://localhost"),
      response: new Response(null, { status: 400 }),
    });

    const state = await createStub({}, fields());

    expect(state).toEqual({ error: "rejected" });
  });
});
