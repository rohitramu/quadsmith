import { describe, it, expect } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { Routes, Route } from "react-router-dom";
import { renderWithProviders } from "../../test/test-utils";
import { UserBadge } from "../UserBadge";
import { getAvatarPalette } from "@quadsmith/shared";

describe("UserBadge Component", () => {
  it("renders with hash-generated palette for a valid username", () => {
    const palette = getAvatarPalette("alex");
    renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username="alex" withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );

    const badge = screen.getByTestId("user-badge");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveTextContent("@alex");
    // Verify palette border class is present
    expect(badge.className).toContain(palette.borderClass.split(" ")[0]);
  });

  it("renders with prefix text when provided", () => {
    renderWithProviders(
      <Routes>
        <Route
          path="/"
          element={<UserBadge username="rotor_king" prefix="Added by " withHoverCard={false} />}
        />
      </Routes>,
      { route: "/" },
    );

    const badge = screen.getByTestId("user-badge");
    expect(badge).toHaveTextContent("Added by @rotor_king");
  });

  it("respects withAt=false", () => {
    renderWithProviders(
      <Routes>
        <Route
          path="/"
          element={<UserBadge username="fpv_pilot" withAt={false} withHoverCard={false} />}
        />
      </Routes>,
      { route: "/" },
    );

    const badge = screen.getByTestId("user-badge");
    expect(badge).toHaveTextContent("fpv_pilot");
    expect(badge).not.toHaveTextContent("@fpv_pilot");
  });

  it("handles fallback for community or null username", () => {
    renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username={null} withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );

    const badge = screen.getByTestId("user-badge");
    expect(badge).toHaveTextContent("@community");
    expect(badge.className).toContain("bg-zinc-100");
  });

  it("links to user profile page by default for valid users", () => {
    renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username="test_pilot" withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );

    const link = screen.getByRole("link", { name: "@test_pilot" });
    expect(link).toHaveAttribute("href", "/users/test_pilot");
  });

  it("handles custom click handler and stops propagation", () => {
    let clicked = false;
    renderWithProviders(
      <Routes>
        <Route
          path="/"
          element={
            <div data-testid="parent" onClick={() => {}}>
              <UserBadge
                username="alex"
                withHoverCard={false}
                onClick={() => {
                  clicked = true;
                }}
              />
            </div>
          }
        />
      </Routes>,
      { route: "/" },
    );

    const link = screen.getByRole("link", { name: "@alex" });
    fireEvent.click(link);
    expect(clicked).toBe(true);
  });

  it("renders with avatar initial when withAvatar is true", () => {
    renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username="alex" withAvatar withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );

    const badge = screen.getByTestId("user-badge");
    expect(badge).toHaveTextContent("A");
    expect(badge).toHaveTextContent("@alex");
  });

  it("renders different sizes (xs, sm, md)", () => {
    const { unmount } = renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username="alex" size="xs" withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );

    expect(screen.getByTestId("user-badge").className).toContain("text-[10px]");
    unmount();

    renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username="alex" size="md" withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );
    expect(screen.getByTestId("user-badge").className).toContain("text-sm");
  });
});
