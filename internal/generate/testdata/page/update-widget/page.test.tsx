import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@/test-utils";
import Page from "./page";

vi.mock("@/lib/api", () => ({
  api: { example: { updateWidget: vi.fn() } },
}));

const params = Promise.resolve({ slug: ["example"], rest: ["example"] });

describe("UpdateWidgetPage", () => {
  it("renders the form", async () => {
    render(await Page({ params }));

    expect(
      screen.getByRole("heading", { level: 1, name: "Update a widget" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Count", { exact: false })).toBeInTheDocument();
    expect(screen.getByLabelText("Email", { exact: false })).toBeInTheDocument();
    expect(screen.getByLabelText("Enabled", { exact: false })).toBeInTheDocument();
    expect(screen.getByLabelText("Note", { exact: false })).toBeInTheDocument();
    expect(screen.getByLabelText("Password", { exact: false })).toBeInTheDocument();
    expect(screen.getByLabelText("Price", { exact: false })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Update a widget" }),
    ).toBeInTheDocument();
  });
});
