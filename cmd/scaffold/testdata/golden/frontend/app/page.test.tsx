import { describe, expect, it } from "vitest";
import { render, screen } from "../test-utils";
import Page from "./page";

describe("Page", () => {
  it("renders the placeholder home page without error", () => {
    render(<Page />);
    expect(
      screen.getByRole("heading", { level: 1, name: "golden-app" }),
    ).toBeInTheDocument();
  });
});
