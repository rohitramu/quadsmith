import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { UserProfilePage } from "../UserProfilePage";
import { Route, Routes } from "react-router-dom";
import { create } from "@bufbuild/protobuf";
import { BuildSchema, type Build } from "../../gen/quadsmith/build_pb";
import {
  mockBuild1,
  mockBuild2,
  mockFrame1,
  mockMotor1,
  mockFC1,
  mockESC1,
  mockPropeller1,
  mockVTX1,
} from "../../test/mocks/fixtures";

const mockPilotBuild: Build = create(BuildSchema, {
  uuid: "01923019-2131-4192-3192-391294812397",
  id: "cinewhoop-indoor-pro",
  name: "CineWhoop Indoor Pro",
  description:
    "A smooth and protected 3.5-inch cinewhoop tuned for high-resolution indoor video capture without prop wash.",
  authorUsername: "fpv_pilot",
  userUuid: "018f0000-0000-7000-0000-000000000099",
  frameUuid: mockFrame1.uuid,
  motorUuid: mockMotor1.uuid,
  flightControllerUuid: mockFC1.uuid,
  electronicSpeedControllerUuids: [mockESC1.uuid],
  propellerUuid: mockPropeller1.uuid,
  videoTransmitterUuid: mockVTX1.uuid,
  receiverUuids: [],
  antennaUuids: [],
  referenceLinks: [],
});

describe("UserProfilePage Component", () => {
  function renderUserProfile(
    route: string = "/users/fpv_pilot",
    builds: Build[] = [mockBuild1, mockBuild2, mockPilotBuild],
  ) {
    return renderWithProviders(
      <Routes>
        <Route path="/users/:id" element={<UserProfilePage />} />
      </Routes>,
      {
        route,
        transportOptions: {
          builds,
        },
      },
    );
  }

  it("renders user profile header card with username, bio, and stats", async () => {
    renderUserProfile("/users/fpv_pilot");

    // Profile Header info
    expect(
      await screen.findByRole("heading", { level: 1, name: "@fpv_pilot" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Freestyle FPV pilot & frame builder/i)).toBeInTheDocument();
    expect(screen.getByText(/Member since/i)).toBeInTheDocument();

    // Stats
    expect(screen.getAllByText("Authored Builds").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Collections").length).toBeGreaterThan(0);
    expect(screen.getByText("Reputation")).toBeInTheDocument();
    expect(screen.getByText("145")).toBeInTheDocument();
  });

  it("renders the larger tabbed section with Builds and Collections tabs", async () => {
    renderUserProfile("/users/fpv_pilot");

    // Tabs
    const tablist = await screen.findByRole("tablist", { name: /user profile content/i });
    expect(tablist).toBeInTheDocument();

    const buildsTab = screen.getByRole("tab", { name: /builds/i });
    const collectionsTab = screen.getByRole("tab", { name: /collections/i });

    expect(buildsTab).toHaveAttribute("aria-selected", "true");
    expect(collectionsTab).toHaveAttribute("aria-selected", "false");
  });

  it("displays builds in the Builds tab and allows scrolling through build cards", async () => {
    renderUserProfile("/users/fpv_pilot");

    // Subheader
    expect(
      await screen.findByRole("heading", { level: 3, name: "Authored Builds" }),
    ).toBeInTheDocument();

    // Build card for fpv_pilot
    expect(await screen.findByText("CineWhoop Indoor Pro")).toBeInTheDocument();
    expect(
      screen.getByText(
        "A smooth and protected 3.5-inch cinewhoop tuned for high-resolution indoor video capture without prop wash.",
      ),
    ).toBeInTheDocument();

    // Link to build profile
    const buildLinks = screen.getAllByRole("link", { name: /cinewhoop indoor pro/i });
    expect(buildLinks.length).toBeGreaterThan(0);
    expect(buildLinks[0]).toHaveAttribute("href", "/builds/cinewhoop-indoor-pro");
  });

  it("switches to Collections tab and displays user's curated collections", async () => {
    const { user } = renderUserProfile("/users/fpv_pilot");

    const collectionsTab = await screen.findByRole("tab", { name: /collections/i });
    await user.click(collectionsTab);

    // Collections tab is now selected
    expect(collectionsTab).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: /builds/i })).toHaveAttribute("aria-selected", "false");

    // Collections panel subheader
    expect(
      await screen.findByRole("heading", { level: 3, name: "Curated Collections" }),
    ).toBeInTheDocument();

    // Curated Collections are visible
    expect(screen.getByText("Liked")).toBeInTheDocument();
    expect(screen.getByText("Cinewhoop Favorites")).toBeInTheDocument();
    expect(screen.getByText("Bando Builds")).toBeInTheDocument();

    // Visibility badges
    expect(screen.getByText("Public")).toBeInTheDocument();
    expect(screen.getAllByText("Private").length).toBeGreaterThan(0);
  });

  it("deep links to Collections tab via URL search params ?tab=collections", async () => {
    renderUserProfile("/users/fpv_pilot?tab=collections");

    const collectionsTab = await screen.findByRole("tab", { name: /collections/i });
    expect(collectionsTab).toHaveAttribute("aria-selected", "true");

    expect(
      await screen.findByRole("heading", { level: 3, name: "Curated Collections" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Cinewhoop Favorites")).toBeInTheDocument();
  });

  it("switches tabs when clicking on profile header stats badges", async () => {
    const { user } = renderUserProfile("/users/fpv_pilot");

    const colBadgeBtn = await screen.findByRole("button", { name: /view collections/i });
    await user.click(colBadgeBtn);

    const collectionsTab = screen.getByRole("tab", { name: /collections/i });
    expect(collectionsTab).toHaveAttribute("aria-selected", "true");
    expect(await screen.findByText("Cinewhoop Favorites")).toBeInTheDocument();

    const buildBadgeBtn = screen.getByRole("button", { name: /view authored builds/i });
    await user.click(buildBadgeBtn);

    const buildsTab = screen.getByRole("tab", { name: /builds/i });
    expect(buildsTab).toHaveAttribute("aria-selected", "true");
    expect(await screen.findByText("CineWhoop Indoor Pro")).toBeInTheDocument();
  });

  it("displays empty state when user has no builds", async () => {
    renderUserProfile("/users/test_pilot_empty");

    // Profile header
    expect(
      await screen.findByRole("heading", { level: 1, name: "@test_pilot_empty" }),
    ).toBeInTheDocument();

    // Empty state in builds tab
    expect(await screen.findByText(/no published builds yet/i)).toBeInTheDocument();
    expect(
      screen.getByText(
        /when @test_pilot_empty creates and publishes quadcopter builds, they will appear here/i,
      ),
    ).toBeInTheDocument();
  });

  it("allows owner to create a collection via the subtle plus button in collections subheader", async () => {
    const { user } = renderUserProfile("/users/fpv_pilot?tab=collections");

    expect(
      await screen.findByRole("heading", { level: 3, name: "Curated Collections" }),
    ).toBeInTheDocument();

    // Subtle plus button in header
    const addColBtn = screen.getByRole("button", { name: /add collection/i });
    expect(addColBtn).toBeInTheDocument();

    await user.click(addColBtn);

    // Modal opens
    expect(
      await screen.findByRole("heading", { level: 3, name: /create collection/i }),
    ).toBeInTheDocument();

    const nameInput = screen.getByRole("textbox", { name: /^name$/i });
    await user.type(nameInput, "New Long Range Rigs");

    const submitBtn = screen.getByRole("button", { name: /create collection/i });
    await user.click(submitBtn);

    // Modal closes
    expect(
      screen.queryByRole("heading", { level: 3, name: /create collection/i }),
    ).not.toBeInTheDocument();
  });

  it("shows Add Collection button in empty state when user has no collections and allows creation", async () => {
    const { user } = renderWithProviders(
      <Routes>
        <Route path="/users/:id" element={<UserProfilePage />} />
      </Routes>,
      {
        route: "/users/fpv_pilot?tab=collections",
        transportOptions: {
          collections: [],
        },
      },
    );

    expect(await screen.findByText(/no collections found/i)).toBeInTheDocument();

    // Prominent "Add Collection" button in empty state
    const addBtn = screen.getByRole("button", { name: /add collection/i });
    expect(addBtn).toBeInTheDocument();

    await user.click(addBtn);

    expect(
      await screen.findByRole("heading", { level: 3, name: /create collection/i }),
    ).toBeInTheDocument();
  });

  it("allows owner to delete a collection using the garbage bin icon on the card", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    const { user } = renderUserProfile("/users/fpv_pilot?tab=collections");

    expect(await screen.findByText("Cinewhoop Favorites")).toBeInTheDocument();

    const deleteBtn = screen.getByRole("button", { name: /delete cinewhoop favorites/i });
    expect(deleteBtn).toBeInTheDocument();

    await user.click(deleteBtn);

    expect(confirmSpy).toHaveBeenCalled();
  });

  it("makes collection cards clickable buttons navigating to collection page without an Explore button", async () => {
    renderUserProfile("/users/fpv_pilot?tab=collections");

    expect(await screen.findByText("Cinewhoop Favorites")).toBeInTheDocument();

    // Verify there is no "Explore" button
    expect(screen.queryByRole("link", { name: /explore/i })).not.toBeInTheDocument();
    expect(screen.queryByText(/explore/i)).not.toBeInTheDocument();

    // Verify the card itself has a clickable link to open the collection
    const cardLink = screen.getByRole("link", { name: /open cinewhoop favorites collection/i });
    expect(cardLink).toBeInTheDocument();
    expect(cardLink).toHaveAttribute("href", "/users/fpv_pilot/collections/cinewhoop-favorites");
  });
});
