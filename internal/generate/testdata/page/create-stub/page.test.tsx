import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@/test-utils";
import Page from "./page";

vi.mock("@/lib/api", () => ({
  api: { example: { createStub: vi.fn() } },
}));

describe("CreateStubPage", () => {
  it("renders the form", () => {
    render(Page());

    expect(
      screen.getByRole("heading", { level: 1, name: "Create a stub" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Name", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Create a stub" }),
    ).toBeInTheDocument();
  });
});
