import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, act, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { UserHoverCard } from "../UserHoverCard";

describe("UserHoverCard Component", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders trigger and displays user hover card with provided profile on hover after delay", async () => {
    const mockProfile = {
      id: "alex",
      avatarUrl: "",
      bio: "FPV drone builder and bando pilot",
      createdAt: { seconds: 1716000000n, nanos: 0 },
      buildCount: 5,
      reputationScore: 120,
    };

    renderWithProviders(
      <UserHoverCard username="alex" profile={mockProfile}>
        <span data-testid="user-trigger">@alex</span>
      </UserHoverCard>,
    );

    const trigger = screen.getByTestId("user-trigger");
    expect(trigger).toBeInTheDocument();

    // Hover card should not be visible before hover
    expect(screen.queryByTestId("user-hover-card")).not.toBeInTheDocument();

    // Fire mouseEnter
    act(() => {
      fireEvent.mouseEnter(trigger);
      vi.advanceTimersByTime(250);
    });

    // Hover card should now be in the document
    const card = screen.getByTestId("user-hover-card");
    expect(card).toBeInTheDocument();

    // Verify user profile information is displayed
    expect(screen.getByRole("link", { name: "@alex" })).toHaveAttribute("href", "/users/alex");
    expect(screen.getByText("FPV drone builder and bando pilot")).toBeInTheDocument();
    expect(screen.getByText(/joined/i)).toBeInTheDocument();
    expect(screen.getByText(/5 builds/i)).toBeInTheDocument();
    expect(screen.getByText(/120 rep/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /view profile/i })).toHaveAttribute(
      "href",
      "/users/alex",
    );

    // Mouse leave with grace period
    act(() => {
      fireEvent.mouseLeave(trigger);
      vi.advanceTimersByTime(200);
    });

    // Card should now be removed
    expect(screen.queryByTestId("user-hover-card")).not.toBeInTheDocument();
  });

  it("strips leading @ from username when querying and navigating", async () => {
    const mockProfile = {
      id: "rotor_king",
      avatarUrl: "",
      bio: "",
      createdAt: { seconds: 1716000000n, nanos: 0 },
      buildCount: 2,
      reputationScore: 45,
    };

    renderWithProviders(
      <UserHoverCard username="@rotor_king" profile={mockProfile}>
        <span data-testid="user-trigger">@rotor_king</span>
      </UserHoverCard>,
    );

    const trigger = screen.getByTestId("user-trigger");

    act(() => {
      fireEvent.mouseEnter(trigger);
      vi.advanceTimersByTime(250);
    });

    expect(screen.getByTestId("user-hover-card")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "@rotor_king" })).toHaveAttribute(
      "href",
      "/users/rotor_king",
    );
  });

  it("keeps hover card open when mouse enters the card within grace period", async () => {
    const mockProfile = {
      id: "fpv_pilot",
      avatarUrl: "",
      bio: "Pro freestyle pilot",
      createdAt: { seconds: 1716000000n, nanos: 0 },
      buildCount: 8,
      reputationScore: 340,
    };

    renderWithProviders(
      <UserHoverCard username="fpv_pilot" profile={mockProfile}>
        <span data-testid="user-trigger">@fpv_pilot</span>
      </UserHoverCard>,
    );

    const trigger = screen.getByTestId("user-trigger");

    act(() => {
      fireEvent.mouseEnter(trigger);
      vi.advanceTimersByTime(250);
    });

    const card = screen.getByTestId("user-hover-card");
    expect(card).toBeInTheDocument();

    // Mouse leaves trigger, but enters card before grace period finishes
    act(() => {
      fireEvent.mouseLeave(trigger);
      vi.advanceTimersByTime(50);
      fireEvent.mouseEnter(card);
      vi.advanceTimersByTime(200);
    });

    // Card should still be open
    expect(screen.getByTestId("user-hover-card")).toBeInTheDocument();

    // Now mouse leaves card
    act(() => {
      fireEvent.mouseLeave(card);
      vi.advanceTimersByTime(200);
    });

    expect(screen.queryByTestId("user-hover-card")).not.toBeInTheDocument();
  });

  it("does not render or open hover card for empty, undefined, or community username", async () => {
    const { unmount } = renderWithProviders(
      <UserHoverCard username="community">
        <span data-testid="community-trigger">@community</span>
      </UserHoverCard>,
    );

    const trigger = screen.getByTestId("community-trigger");
    act(() => {
      fireEvent.mouseEnter(trigger);
      vi.advanceTimersByTime(300);
    });

    expect(screen.queryByTestId("user-hover-card")).not.toBeInTheDocument();
    unmount();

    // With null/undefined username
    renderWithProviders(
      <UserHoverCard username={null}>
        <span data-testid="null-trigger">Unknown</span>
      </UserHoverCard>,
    );

    const nullTrigger = screen.getByTestId("null-trigger");
    act(() => {
      fireEvent.mouseEnter(nullTrigger);
      vi.advanceTimersByTime(300);
    });

    expect(screen.queryByTestId("user-hover-card")).not.toBeInTheDocument();
  });

  it("shows loading skeleton while profile is being fetched asynchronously", async () => {
    renderWithProviders(
      <UserHoverCard username="alex">
        <span data-testid="async-trigger">@alex</span>
      </UserHoverCard>,
    );

    const trigger = screen.getByTestId("async-trigger");
    act(() => {
      fireEvent.mouseEnter(trigger);
      vi.advanceTimersByTime(250);
    });

    // Card mounts immediately and shows pulse animation while loading
    const card = screen.getByTestId("user-hover-card");
    expect(card).toBeInTheDocument();
    expect(card.querySelector(".animate-pulse")).toBeInTheDocument();
  });
});
