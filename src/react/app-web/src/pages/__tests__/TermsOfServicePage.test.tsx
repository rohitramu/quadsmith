import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { TermsOfServicePage } from "../TermsOfServicePage";
import { AppRoutes } from "../../App";

describe("TermsOfServicePage Component", () => {
  it("renders Terms of Service, FPV safety disclaimers, Experimental Build Evaluator section, and Privacy summary without top table of contents links", () => {
    renderWithProviders(<TermsOfServicePage />, { route: "/terms" });

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: /terms of service & disclaimers/i,
      }),
    ).toBeInTheDocument();

    // Top table of contents / quick-section links should not be rendered
    expect(screen.queryByRole("button", { name: /safety disclaimers/i })).not.toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: /experimental build evaluator/i }),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /^terms of service$/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /data & privacy/i })).not.toBeInTheDocument();

    // Critical safety warning callout
    expect(screen.getByRole("note", { name: /critical safety warning/i })).toBeInTheDocument();

    // Section 1: Safety Disclaimers
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /^safety disclaimers$/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/electrical & lipo battery safety/i)).toBeInTheDocument();
    expect(screen.getByText(/mechanical & flight safety/i)).toBeInTheDocument();
    expect(screen.getByText(/local rules & regulations/i)).toBeInTheDocument();
    expect(screen.getByText(/catalog data & hardware revisions/i)).toBeInTheDocument();

    // Section 2: Experimental Build Evaluator & Compatibility Engine
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /experimental build evaluator & compatibility engine/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/provided “as is” without warranty:/i)).toBeInTheDocument();
    expect(screen.queryByText(/featured on build profile pages/i)).not.toBeInTheDocument();

    // Section 3: Terms of Service (broad paragraphs, no numbered sub-clauses)
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /^terms of service$/i,
      }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/1\. acceptance of terms/i)).not.toBeInTheDocument();
    expect(screen.getByText(/indemnify, defend, and hold harmless quadsmith/i)).toBeInTheDocument();
    expect(
      screen.getByText(/modify, suspend, or discontinue any part of the service/i),
    ).toBeInTheDocument();

    // Section 4: Data & Privacy (broad privacy notice, no export/deletion specifics)
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /data & privacy/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/no internet transmission or electronic storage is completely secure/i),
    ).toBeInTheDocument();
    expect(screen.queryByText(/\.zip/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/30-day/i)).not.toBeInTheDocument();
  });

  it("renders standalone header when accessed by unauthenticated users via /terms", async () => {
    renderWithProviders(<AppRoutes />, {
      route: "/terms",
      transportOptions: { isAuthenticated: false },
    });

    expect(
      await screen.findByRole("heading", {
        level: 1,
        name: /terms of service & disclaimers/i,
      }),
    ).toBeInTheDocument();
    const signInLinks = screen.getAllByRole("link", { name: /sign in/i });
    expect(signInLinks.length).toBeGreaterThanOrEqual(1);
    expect(signInLinks[0]).toHaveAttribute("href", "/login");
  });

  it("renders inside Layout when accessed by authenticated users via /terms", async () => {
    renderWithProviders(<AppRoutes />, {
      route: "/terms",
      transportOptions: { isAuthenticated: true },
    });

    expect(
      await screen.findByRole("heading", {
        level: 1,
        name: /terms of service & disclaimers/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Sidebar" })).toBeInTheDocument();
  });
});
