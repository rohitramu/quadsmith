import { describe, it, expect } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { Routes, Route } from "react-router-dom";
import { renderWithProviders } from "../../test/test-utils";
import { UserBadge } from "../UserBadge";
import { getAvatarPalette } from "@quadsmith/shared";

describe("UserBadge Component", () => {
  it("renders with hash-generated palette for a valid username with signature border and highlight", () => {
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
    expect(badge.className).toContain(palette.borderClass.split(" ")[0]);
    expect(badge.className).toContain("shadow-[0_0_6px_2.5px_rgba(39,39,42,0.55)]");
    expect(badge.className).toContain("font-bold");
    expect(badge.className).toContain(palette.bgClass.split(" ")[0]);
  });

  it("renders with prefix text when provided, outside the pill badge", () => {
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
    expect(badge).toHaveTextContent("@rotor_king");
    expect(badge).not.toHaveTextContent("Added by");

    const container = screen.getByTestId("user-badge-container");
    expect(container).toHaveTextContent("Added by @rotor_king");
    expect(container.firstChild).toHaveTextContent("Added by");
    expect(container.firstChild).toHaveClass("text-zinc-500");
  });

  it("renders with gold border and Author title when isAuthor is true", () => {
    renderWithProviders(
      <Routes>
        <Route
          path="/"
          element={<UserBadge username="rotor_king" isAuthor={true} withHoverCard={false} />}
        />
      </Routes>,
      { route: "/" },
    );

    const badge = screen.getByTestId("user-badge");
    expect(badge).toHaveAttribute("title", "Author");
    expect(badge.className).toContain("border-[#ca8a04]");
    expect(badge.className).toContain("dark:border-[#fbbf24]");
    expect(badge.className).toContain("shadow-[0_0_6px_2.5px_rgba(39,39,42,0.55)]");
  });

  it("renders with signature border, halo shadow, bold font, and You title when isSelf is true", () => {
    const palette = getAvatarPalette("alex");
    renderWithProviders(
      <Routes>
        <Route
          path="/"
          element={<UserBadge username="alex" isSelf={true} size="sm" withHoverCard={false} />}
        />
      </Routes>,
      { route: "/" },
    );

    const badge = screen.getByTestId("user-badge");
    expect(badge).toHaveAttribute("title", "You");
    expect(badge.className).toContain("font-bold");
    expect(badge.className).toContain("px-2.5");
    expect(badge.className).toContain("py-1");
    expect(badge.className).toContain(palette.borderClass.split(" ")[0]);
    expect(badge.className).toContain("shadow-[0_0_6px_2.5px_rgba(39,39,42,0.55)]");
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

  it("renders different sizes (xs, sm, md, lg)", () => {
    const { unmount } = renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username="alex" size="xs" withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );

    expect(screen.getByTestId("user-badge").className).toContain("text-[10px]");
    unmount();

    const { unmount: unmountMd } = renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username="alex" size="md" withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );
    expect(screen.getByTestId("user-badge").className).toContain("text-sm");
    unmountMd();

    renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username="alex" size="lg" withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );
    expect(screen.getByTestId("user-badge").className).toContain("text-lg");
  });

  it("renders as non-clickable span when to is null", () => {
    renderWithProviders(
      <Routes>
        <Route path="/" element={<UserBadge username="alex" to={null} withHoverCard={false} />} />
      </Routes>,
      { route: "/" },
    );

    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByTestId("user-badge")).toHaveTextContent("@alex");
  });
});
