import { describe, expect, it, vi } from "vitest";
import { api } from "@/lib/api";
import { render, screen } from "@/test-utils";
import Page from "./page";

vi.mock("@/lib/api", () => ({
  api: { users: { getCurrentUser: vi.fn() } },
}));

const getCurrentUser = vi.mocked(api.users.getCurrentUser);

describe("GetCurrentUserPage", () => {
  it("renders the response", async () => {
    getCurrentUser.mockResolvedValue({
      data: { email: "email value", emailVerified: true, id: "id value", permissions: [], role: "role value" },
      error: undefined,
      request: new Request("http://localhost"),
      response: new Response(),
    });

    render(await Page());

    expect(getCurrentUser).toHaveBeenCalledWith();
    expect(
      screen.getByRole("heading", { level: 1, name: "Get the current user" }),
    ).toBeInTheDocument();
    expect(screen.getByText("email value")).toBeInTheDocument();
    expect(screen.getByText("id value")).toBeInTheDocument();
    expect(screen.getByText("role value")).toBeInTheDocument();
  });

  it("shows the error detail", async () => {
    getCurrentUser.mockResolvedValue({
      data: undefined,
      error: { detail: "rejected" },
      request: new Request("http://localhost"),
      response: new Response(null, { status: 400 }),
    });

    render(await Page());

    expect(screen.getByRole("alert")).toHaveTextContent("rejected");
  });
});
