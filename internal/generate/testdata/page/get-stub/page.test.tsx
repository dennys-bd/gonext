import { describe, expect, it, vi } from "vitest";
import { api } from "@/lib/api";
import { render, screen } from "@/test-utils";
import Page from "./page";

vi.mock("@/lib/api", () => ({
  api: { example: { getStub: vi.fn() } },
}));

const getStub = vi.mocked(api.example.getStub);
const params = Promise.resolve({ id: "example" });

describe("GetStubPage", () => {
  it("renders the response", async () => {
    getStub.mockResolvedValue({
      data: { createdAt: "createdAt value", id: "id value", name: "name value" },
      error: undefined,
      request: new Request("http://localhost"),
      response: new Response(),
    });

    render(await Page({ params }));

    expect(getStub).toHaveBeenCalledWith({ path: { id: "example" } });
    expect(
      screen.getByRole("heading", { level: 1, name: "Get a stub by id" }),
    ).toBeInTheDocument();
    expect(screen.getByText("createdAt value")).toBeInTheDocument();
    expect(screen.getByText("id value")).toBeInTheDocument();
    expect(screen.getByText("name value")).toBeInTheDocument();
  });

  it("shows the error detail", async () => {
    getStub.mockResolvedValue({
      data: undefined,
      error: { detail: "rejected" },
      request: new Request("http://localhost"),
      response: new Response(null, { status: 400 }),
    });

    render(await Page({ params }));

    expect(screen.getByRole("alert")).toHaveTextContent("rejected");
  });
});
