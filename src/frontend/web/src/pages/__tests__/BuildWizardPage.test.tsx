import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import { BuildWizardPage } from "../BuildWizardPage";
import { Route, Routes } from "react-router-dom";

describe("BuildWizardPage Component", () => {
  function renderWizard() {
    return renderWithProviders(
      <Routes>
        <Route path="/builds/new" element={<BuildWizardPage />} />
        <Route
          path="/builds/:buildId"
          element={<div data-testid="build-profile-page">Build Saved Profile</div>}
        />
      </Routes>,
      { route: "/builds/new" },
    );
  }

  it("renders with initial empty state: 0 parts selected, 0.0g dry weight, Stage 1 active", async () => {
    renderWizard();

    // Check title & banner
    expect(
      await screen.findByRole("heading", { level: 1, name: "Design Custom Drone" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Interactive Build Wizard/i)).toBeInTheDocument();

    // Stage 1 active, stages 2-4 disabled
    expect(screen.getByText("Airframe & Propulsion")).toBeInTheDocument();
    expect(screen.getByText("0/3")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /stage 2/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /stage 3/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /stage 4/i })).toBeDisabled();
    expect(screen.getAllByText("Required").length).toBe(3);

    // Initial dry weight is 0.0g
    expect(screen.getByText("0.0g")).toBeInTheDocument();
  });

  it("allows searching for frames and filtering products in real time", async () => {
    const user = userEvent.setup();
    renderWizard();

    const searchInput = await screen.findByPlaceholderText(
      /Search frames by name, brand, geometry/i,
    );
    expect(searchInput).toBeInTheDocument();

    // Type non-matching search term
    await user.type(searchInput, "nonexistent-frame-xyz");
    expect(
      await screen.findByText(/No compatible frames matching your search/i),
    ).toBeInTheDocument();

    // Clear search
    await user.clear(searchInput);
    expect(
      screen.queryByText(/No compatible frames matching your search/i),
    ).not.toBeInTheDocument();
  });

  it("allows selecting Stage 1 parts, dynamically updates motor count, and unlocks Stage 2", async () => {
    const user = userEvent.setup();
    renderWizard();

    // 1A. Select Frame
    const frameCard = await screen.findByText("Master 5 V2");
    await user.click(frameCard);

    // 1B. Select Motor
    const motorCard = await screen.findByText("ECO II 2207");
    await user.click(motorCard);

    // 1C. Select Propeller
    const propCard = await screen.findByText("Hurricane 51433");
    await user.click(propCard);

    // Dry weight should now be > 0
    await waitFor(() => {
      expect(screen.queryByText("0.0g")).not.toBeInTheDocument();
    });

    // Next button should be enabled
    const nextBtn = screen.getByRole("button", { name: /Next: Flight Electronics/i });
    expect(nextBtn).toBeEnabled();

    // Advance to Stage 2
    await user.click(nextBtn);
    expect(await screen.findByText("2A. Flight Controller")).toBeInTheDocument();
  });

  it("handles conditional ESC selection: standalone FC requires ESC, AIO FC allows built-in ESC", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Select Stage 1 parts
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));

    // Go to Stage 2
    await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));

    // Select standalone FC (F405 V4)
    await user.click(await screen.findByText("F405 V4 FC"));
    expect(screen.getByText("Requires separate ESC")).toBeInTheDocument();

    // In-built FC ESC card should be disabled
    expect(
      screen.getByText("Use In-built FC ESC").closest(".cursor-not-allowed"),
    ).toBeInTheDocument();
    expect(screen.getByText("Requires AIO FC with internal ESC")).toBeInTheDocument();

    // Select AIO FC (Integrated FC AIO)
    await user.click(screen.getByText("Integrated FC AIO"));
    expect(screen.getByText(/✓ Available on selected FC/i)).toBeInTheDocument();

    // Now In-built FC ESC card is clickable
    await user.click(screen.getByText("Use In-built FC ESC"));
    expect(
      screen.getByText("Use In-built FC ESC").closest(".border-emerald-500"),
    ).toBeInTheDocument();
    expect(screen.getAllByText("Selected").length).toBeGreaterThanOrEqual(1);
  });

  it("handles Stage 3 optional components and 'Set All to None'", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Complete Stage 1
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));

    // Complete Stage 2
    await user.click(await screen.findByText("F405 V4 FC"));
    await user.click(await screen.findByText("SpeedyBee 50A 4-in-1 ESC"));
    await user.click(await screen.findByText("Crossfire Nano RX"));

    // Advance to Stage 3
    const nextBtn = screen.getByRole("button", { name: /Next: Vision & Navigation/i });
    expect(nextBtn).toBeEnabled();
    await user.click(nextBtn);

    expect(await screen.findByText(/All components in Stage 3 are optional/i)).toBeInTheDocument();
    expect(screen.getByText("3D. GPS Receiver & Compass")).toBeInTheDocument();

    // Click "Set All to None"
    const skipAllBtn = screen.getByRole("button", { name: /Set All to None/i });
    await user.click(skipAllBtn);

    // Verify explicit None selections
    expect(screen.getByText("No GPS (None)")).toBeInTheDocument();
    expect(screen.getByText("No VTX (None)")).toBeInTheDocument();
    expect(screen.getByText("No Camera (None)")).toBeInTheDocument();
    expect(screen.getByText("No Antenna (None)")).toBeInTheDocument();
  });

  it("allows interactive payload adjustment and preset selection in Evaluator", async () => {
    const user = userEvent.setup();
    renderWizard();

    const payloadInput = (await screen.findByPlaceholderText("0")) as HTMLInputElement;
    expect(payloadInput.value).toBe("0");

    // Click + button to add 10g
    const plusBtn = screen.getByRole("button", { name: /Add 10g/i });
    await user.click(plusBtn);
    expect(screen.getByText("+10g")).toBeInTheDocument();

    // Click GoPro preset (+133g)
    const goproBtn = screen.getByRole("button", { name: /GoPro \(\+133g\)/i });
    await user.click(goproBtn);
    expect(screen.getByText("+133g")).toBeInTheDocument();

    // Click Bare preset (0g)
    const bareBtn = screen.getByRole("button", { name: /Bare \(0g\)/i });
    await user.click(bareBtn);
    expect(screen.getByText("+0g")).toBeInTheDocument();
  });

  it("allows reviewing BOM in Stage 4 and saving the build to PostgreSQL", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Use 5" Freestyle preset to quickly populate all parts
    const presetBtn = await screen.findByRole("button", { name: /5" Freestyle Preset/i });
    await user.click(presetBtn);

    // Navigate to Stage 4 (Review & Save)
    const stage4Tab = screen.getByRole("button", { name: /Review & Save/i });
    await user.click(stage4Tab);

    // Check BOM items
    expect(await screen.findByText(/Bill of Materials \(BOM\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Motors: ECO II 2207 \(4x\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Flight Controller: F405 V4 FC/i)).toBeInTheDocument();

    // Click Save Build
    const saveBtn = screen.getByRole("button", { name: /Create & Save Build to Database/i });
    await user.click(saveBtn);

    // Should navigate to saved build profile
    expect(await screen.findByTestId("build-profile-page")).toBeInTheDocument();
  });
});
