import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@/test-utils";
import Page from "./page";

vi.mock("@/lib/api", () => ({
  api: { users: { logoutUser: vi.fn() } },
}));

describe("LogoutUserPage", () => {
  it("renders the form", () => {
    render(Page());

    expect(
      screen.getByRole("heading", { level: 1, name: "Log out" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Log out" }),
    ).toBeInTheDocument();
  });
});
