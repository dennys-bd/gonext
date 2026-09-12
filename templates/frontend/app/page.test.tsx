import { describe, expect, it, vi } from "vitest";
import { api } from "@/lib/api";
import { render, screen } from "../test-utils";
import Page from "./page";

vi.mock("@/lib/api", () => ({
  api: { users: { getCurrentUser: vi.fn(), logoutUser: vi.fn() } },
}));

const getCurrentUser = vi.mocked(api.users.getCurrentUser);

describe("Page", () => {
  it("greets the signed-in user and offers to log out", async () => {
    getCurrentUser.mockResolvedValue({
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

    render(await Page());

    expect(
      screen.getByRole("heading", { level: 1, name: "[PROJECT-NAME]" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { level: 2, name: "ada@example.com" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Log out" })).toBeInTheDocument();
  });

  it("links to the login page when signed out", async () => {
    getCurrentUser.mockResolvedValue({
      data: undefined,
      error: { detail: "unauthorized", status: 401 },
      request: new Request("http://localhost"),
      response: new Response(null, { status: 401 }),
    });

    render(await Page());

    expect(
      screen.getByRole("heading", { level: 1, name: "[PROJECT-NAME]" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Log in" })).toHaveAttribute(
      "href",
      "/login",
    );
    expect(screen.queryByRole("button", { name: "Log out" })).toBeNull();
  });
});
