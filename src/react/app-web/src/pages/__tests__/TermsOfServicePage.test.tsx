import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { TermsOfServicePage } from "../TermsOfServicePage";
import { AppRoutes } from "../../App";

describe("TermsOfServicePage Component", () => {
  it("renders Terms of Service, FPV safety disclaimers, Experimental Build Evaluator section, and Privacy summary", () => {
    renderWithProviders(<TermsOfServicePage />, { route: "/terms" });

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: /terms of service & disclaimers/i,
      }),
    ).toBeInTheDocument();

    // Critical safety warning callout
    expect(screen.getByRole("note", { name: /critical safety warning/i })).toBeInTheDocument();

    // Section 1: Safety & Engineering Disclaimers
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /safety & engineering disclaimers/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/electrical & lipo battery safety/i)).toBeInTheDocument();
    expect(screen.getByText(/mechanical & flight safety/i)).toBeInTheDocument();
    expect(screen.getByText(/rf spectrum & aviation compliance/i)).toBeInTheDocument();
    expect(screen.getByText(/catalog data & hardware revisions/i)).toBeInTheDocument();

    // Section 2: Experimental Build Evaluator & Compatibility Engine
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /experimental build evaluator & compatibility engine/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/idealized aerodynamic & electrical assumptions/i)).toBeInTheDocument();

    // Section 3: Terms of Service
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /^terms of service$/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/1\. acceptance of terms/i)).toBeInTheDocument();
    expect(
      screen.getByText(/3\. limitation of liability & assumption of risk/i),
    ).toBeInTheDocument();

    // Section 4: Data & Privacy
    expect(
      screen.getByRole("heading", {
        level: 2,
        name: /data & privacy/i,
      }),
    ).toBeInTheDocument();
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
