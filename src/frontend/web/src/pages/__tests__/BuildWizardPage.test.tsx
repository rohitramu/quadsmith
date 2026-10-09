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

  it("handles conditional ESC and Receiver selection: standalone FC requires separate components, AIO FC allows integrated components with rich specs", async () => {
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

    // Integrated FC ESC card should be disabled
    expect(
      screen.getByText("Use Integrated FC ESC").closest(".cursor-not-allowed"),
    ).toBeInTheDocument();
    expect(screen.getByText("Requires AIO FC with integrated ESC")).toBeInTheDocument();

    // Integrated FC Receiver card should be disabled
    expect(
      screen.getByText("Use Integrated FC Receiver").closest(".cursor-not-allowed"),
    ).toBeInTheDocument();
    expect(screen.getByText(/Requires FC with integrated receiver/i)).toBeInTheDocument();

    // Select AIO FC (Integrated FC AIO)
    await user.click(screen.getByText("Integrated FC AIO"));
    expect(screen.getAllByText(/✓ Available on selected FC/i).length).toBeGreaterThanOrEqual(1);

    // Now Integrated FC ESC card is clickable and displays rich specs
    expect(screen.getByText("BetaFPV Integrated 20A ESC")).toBeInTheDocument();
    expect(screen.getByText("20A")).toBeInTheDocument();
    expect(screen.getByText(/4x Motors • BLHeli_S/i)).toBeInTheDocument();
    await user.click(screen.getByText("Use Integrated FC ESC"));
    expect(
      screen.getByText("Use Integrated FC ESC").closest(".border-emerald-500"),
    ).toBeInTheDocument();

    // Integrated FC Receiver card is clickable and displays rich specs (protocol, frequency band, telemetry)
    expect(screen.getByText("BetaFPV Integrated ELRS 2.4GHz RX")).toBeInTheDocument();
    expect(screen.getByText("ExpressLRS")).toBeInTheDocument();
    expect(screen.getByText(/2.4 GHz • Telemetry/i)).toBeInTheDocument();
    await user.click(screen.getByText("Use Integrated FC Receiver"));
    expect(
      screen.getByText("Use Integrated FC Receiver").closest(".border-emerald-500"),
    ).toBeInTheDocument();
  });

  it("handles Stage 2 RX antenna & GPS and Stage 3 Video optional components with 'Set All to None'", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Complete Stage 1
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));

    // Verify Stage 2 sections: 2A FC, 2B ESC, 2C RX, 2D RX Antenna, 2E GPS
    expect(await screen.findByText("2A. Flight Controller")).toBeInTheDocument();
    expect(screen.getByText("2B. Electronic Speed Controller (ESC)")).toBeInTheDocument();
    expect(screen.getByText("2C. Radio Control Receiver")).toBeInTheDocument();
    expect(screen.getByText("2D. Radio Receiver Antenna(s)")).toBeInTheDocument();
    expect(screen.getByText("2E. GPS Receiver & Compass")).toBeInTheDocument();

    // Select Stage 2 components
    await user.click(screen.getByText("F405 V4 FC"));
    await user.click(screen.getByText("SpeedyBee 50A 4-in-1 ESC"));
    await user.click(screen.getByText("Crossfire Nano RX"));

    // Select RX Antenna & test diversity toggle
    const rxAntCard = screen.getByText("RadioMaster T-Antenna 2.4GHz");
    await user.click(rxAntCard);
    expect(screen.getByText("Antenna Quantity:")).toBeInTheDocument();
    const divBtn = screen.getByRole("button", { name: /2x Diversity/i });
    await user.click(divBtn);
    expect(screen.getByText(/2x Selected \(Diversity\)/i)).toBeInTheDocument();

    // Select GPS in Stage 2
    const gpsCard = await screen.findByText(/M8Q-5883/i);
    await user.click(gpsCard);

    // Advance to Stage 3 (Video)
    const nextBtn = screen.getByRole("button", { name: /Next: Video/i });
    expect(nextBtn).toBeEnabled();
    await user.click(nextBtn);

    // Verify Stage 3 Video sections: 3A VTX, 3B Camera, 3C Video Transmitter Antenna(s)
    expect(
      await screen.findByText(/All video components in Stage 3 are optional/i),
    ).toBeInTheDocument();
    expect(screen.getByText("3A. Video Transmitter (VTX)")).toBeInTheDocument();
    expect(screen.getByText("3B. FPV Camera")).toBeInTheDocument();
    expect(screen.getByText("3C. Video Transmitter Antenna(s)")).toBeInTheDocument();

    // Test VTX Antenna selection & dual toggle
    const vtxAntCard = await screen.findByText(/Lollipop 4/i);
    await user.click(vtxAntCard);
    expect(screen.getByRole("button", { name: /2x Dual/i })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /2x Dual/i }));
    expect(screen.getByText(/2x Selected \(Dual\)/i)).toBeInTheDocument();

    // Click "Set All to None (Line-of-Sight)"
    const skipAllBtn = screen.getByRole("button", { name: /Set All to None/i });
    await user.click(skipAllBtn);

    // Verify explicit None selections in Stage 3
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
