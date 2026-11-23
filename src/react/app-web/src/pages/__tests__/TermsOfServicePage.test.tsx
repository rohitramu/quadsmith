import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { TermsOfServicePage } from "../TermsOfServicePage";
import { AppRoutes } from "../../App";

describe("TermsOfServicePage Component", () => {
  it("renders Terms of Service, FPV safety disclaimers, Build Evaluator section, and Privacy summary without top badges or table of contents links", () => {
    renderWithProviders(<TermsOfServicePage />, { route: "/terms" });

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: /terms of service & disclaimers/i,
      }),
    ).toBeInTheDocument();

    // Top Legal & Safety Documentation pill and table of contents links should not be rendered
    expect(screen.queryByText(/legal & safety documentation/i)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /safety disclaimers/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /build evaluator/i })).not.toBeInTheDocument();
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

    // Section 2: Build Evaluator & Compatibility Engine (no "Experimental" labels)
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /^build evaluator & compatibility engine$/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/provided “as is” without warranty:/i)).toBeInTheDocument();
    expect(screen.queryByText(/experimental/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/featured on build profile pages/i)).not.toBeInTheDocument();

    // Section 3: Combined Terms of Service & Privacy
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /^terms of service & privacy$/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", {
        level: 2,
        name: /^data & privacy$/i,
      }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/1\. acceptance of terms/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/indemnify/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/license/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/without notice or liability/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/operate, secure, and improve/i)).not.toBeInTheDocument();
    expect(
      screen.queryByText(/reasonable measures to protect your information/i),
    ).not.toBeInTheDocument();
    expect(
      screen.getByText(/give quadsmith permission to use and display that content/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/at any time without notice\./i)).toBeInTheDocument();
    expect(screen.getByText(/to operate and improve the service/i)).toBeInTheDocument();
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
