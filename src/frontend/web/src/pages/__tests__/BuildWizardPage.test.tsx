import { describe, it, expect, vi } from "vitest";
import { screen, waitFor, within, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import { BuildWizardPage, rxHasIntegratedAntenna, getVtxAntennaCount } from "../BuildWizardPage";
import { Route, Routes, useParams } from "react-router-dom";
import { wasmEngine } from "../../lib/wasmEngine";
import {
  mockBuild1,
  mockBuild2,
  mockReceiver1,
  mockReceiverCeramic,
} from "../../test/mocks/fixtures";
import type { RenderWithProvidersOptions } from "../../test/test-utils";

function SavedProfile() {
  const { buildId } = useParams();
  return (
    <div data-testid="build-profile-page">
      Build Saved Profile: <span data-testid="saved-build-id">{buildId}</span>
    </div>
  );
}

describe("BuildWizardPage Component", () => {
  function renderWizard(options: RenderWithProvidersOptions = {}) {
    return renderWithProviders(
      <Routes>
        <Route path="/builds/new" element={<BuildWizardPage />} />
        <Route path="/builds/:buildId" element={<SavedProfile />} />
      </Routes>,
      { route: "/builds/new", ...options },
    );
  }

  it("renders with initial empty state: Stage 0 active with baseline options", async () => {
    renderWizard();

    // Check title & banner
    expect(
      await screen.findByRole("heading", { level: 1, name: "Design Custom Drone" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Interactive Build Wizard/i)).toBeInTheDocument();

    // Stage 0 active, Stage 1 unlocked (0/3), stages 2-4 disabled
    expect(screen.getByText("Template Selection")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /stage 1/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /stage 2/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /stage 3/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /stage 4/i })).toBeDisabled();

    // Stage 0 baseline options
    expect(screen.getByText("Choose Starting Baseline")).toBeInTheDocument();
    expect(screen.queryByText("Start from Scratch")).not.toBeInTheDocument();
    // Initial dry weight is 0.0g across evaluator and selected parts list
    expect(screen.getAllByText("0.0g").length).toBeGreaterThanOrEqual(1);

    // Selected Parts section under Evaluator
    expect(screen.getByText("Selected Parts")).toBeInTheDocument();
    expect(screen.getByText("0 / 11 Parts")).toBeInTheDocument();
  });

  it("allows searching for frames and filtering products in real time", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Stage 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

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

    // Advance to Stage 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

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

    // Advance to Stage 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

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
    expect(screen.getAllByText("ExpressLRS").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/2.4 GHz • Telemetry/i)).toBeInTheDocument();
    await user.click(screen.getByText("Use Integrated FC Receiver"));
    expect(
      screen.getByText("Use Integrated FC Receiver").closest(".border-emerald-500"),
    ).toBeInTheDocument();
  });

  it("handles Stage 2 RX antenna & GPS and Stage 3 Video optional components with 'Set All to None'", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Stage 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

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

    // Test VTX Antenna selection & automatic antenna derivation
    const vtxAntCard = await screen.findByText(/Lollipop 4/i);
    await user.click(vtxAntCard);
    expect(screen.getByText(/Single Antenna \(Required by VTX\)/i)).toBeInTheDocument();
    expect(screen.getByText("1x")).toBeInTheDocument();

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

  it("allows selecting a template in Stage 0, reviewing BOM in Stage 4, and saving the build to PostgreSQL", async () => {
    const user = userEvent.setup();
    renderWizard();

    // In Stage 0, select "Bando Basher 5 inch" template
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);

    // Template should be active and Stages 1-4 unlocked
    expect(screen.getByText("Template Active")).toBeInTheDocument();

    // All stages 0 to 4 in the stage bar should become marked as Done
    const stageNav = screen.getByRole("navigation", { name: "Build Stages" });
    const stageTabs = within(stageNav).getAllByRole("button");
    expect(stageTabs).toHaveLength(5);
    for (const tab of stageTabs) {
      expect(tab).toHaveTextContent("Done");
    }

    // Navigate to Stage 4 (Review & Save)
    const stage4Tab = within(stageNav).getByRole("button", { name: /Stage 4/i });
    expect(stage4Tab).toBeEnabled();
    await user.click(stage4Tab);

    // Check BOM items populated from template
    expect(await screen.findByText(/Bill of Materials \(BOM\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Motors: ECO II 2207 \(4x\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Flight Controller: F405 V4 FC/i)).toBeInTheDocument();

    // Click Save Build
    const saveBtn = screen.getByRole("button", { name: /Create & Save Build to Database/i });
    await user.click(saveBtn);

    // Should navigate to saved build profile with auto-incremented -copy1 suffix
    expect(await screen.findByTestId("build-profile-page")).toBeInTheDocument();
    expect(screen.getByTestId("saved-build-id")).toHaveTextContent("bando-basher-5-inch-copy1");
  });

  it("auto-increments copy suffix when saving duplicate builds", async () => {
    const user = userEvent.setup();
    const copy1Build = { ...mockBuild1, id: "bando-basher-5-inch-copy1", uuid: "copy1-uuid" };
    renderWizard({
      transportOptions: {
        builds: [mockBuild1, copy1Build, mockBuild2],
      },
    });

    // Select template in Stage 0
    const templateCards = await screen.findAllByText("Bando Basher 5 inch");
    await user.click(templateCards[0]);

    // Navigate to Stage 4 (Review & Save)
    const stageNav = screen.getByRole("navigation", { name: "Build Stages" });
    const stage4Tab = within(stageNav).getByRole("button", { name: /Stage 4/i });
    await user.click(stage4Tab);

    // Change build name to one that generates -copy1
    const nameInput = screen.getByPlaceholderText(/e\.g\. My Freestyle 5-Inch/i);
    await user.clear(nameInput);
    await user.type(nameInput, "Bando Basher 5 inch copy1");

    // Click Save Build
    const saveBtn = screen.getByRole("button", { name: /Create & Save Build to Database/i });
    await user.click(saveBtn);

    // Should auto-detect -copy1 and increment to -copy2
    expect(await screen.findByTestId("build-profile-page")).toBeInTheDocument();
    expect(screen.getByTestId("saved-build-id")).toHaveTextContent("bando-basher-5-inch-copy2");
  });

  it("resets all wizard selections and returns to Stage 0 on Reset button click", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Select template in Stage 0
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);
    expect(screen.getByText("Template Active")).toBeInTheDocument();

    // Advance to Stage 1
    await user.click(screen.getByRole("button", { name: /Next: Airframe & Propulsion/i }));
    expect(screen.getByText("1A. Frame Chassis")).toBeInTheDocument();

    // Click Reset button in the Reset Wizard card
    const resetBtn = screen.getByRole("button", { name: /Reset/i });
    await user.click(resetBtn);

    // Should return to Stage 0 baseline selection
    expect(screen.getByText("Choose Starting Baseline")).toBeInTheDocument();
    expect(screen.getAllByText("0.0g").length).toBeGreaterThanOrEqual(1);
  });

  it("displays dynamic CEL filter when physics engine is active and allows toggling compatible parts", async () => {
    vi.spyOn(wasmEngine, "getIsReady").mockReturnValue(true);
    vi.spyOn(wasmEngine, "generateCelFilter").mockImplementation((target: string) => {
      if (target === "propellers") return "diameter_mm <= 130.0";
      return "";
    });

    const user = userEvent.setup();
    renderWizard();

    // Advance to Stage 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

    // Select Frame (Master 5 V2 - 130mm max prop size)
    await user.click(await screen.findByText("Master 5 V2"));

    // Check that the dynamic CEL filter banner is rendered for propellers (in Step 1C and Sidebar)
    const filterElements = await screen.findAllByText("diameter_mm <= 130.0");
    expect(filterElements.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Compatibility Filter:")).toBeInTheDocument();

    // Toggle off Compatible Only
    const compatibleCheckbox = screen.getByLabelText("Compatible Only");
    expect(compatibleCheckbox).toBeChecked();
    await user.click(compatibleCheckbox);
    expect(compatibleCheckbox).not.toBeChecked();
  });

  it("renders the selected parts list under the build evaluator and updates when components are selected", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Verify Selected Parts card under Evaluator
    expect(screen.getByText("Selected Parts")).toBeInTheDocument();
    expect(screen.getByText("0 / 11 Parts")).toBeInTheDocument();
    expect(screen.getByText("BOM Total Dry Weight:")).toBeInTheDocument();

    // Verify Stage 1 items are unlocked initially
    expect(screen.getByTitle("Go to Stage 1: Frame selection")).toBeEnabled();
    expect(screen.getByTitle("Go to Stage 1: Motor selection")).toBeEnabled();
    expect(screen.getByTitle("Go to Stage 1: Propeller selection")).toBeEnabled();

    // Verify Stage 2 & 3 items are locked initially with disabled buttons
    const fcPartButton = screen.getByTitle(
      "Stage 2 is locked: Flight Controller (Complete Stage 1 first)",
    );
    expect(fcPartButton).toBeDisabled();
    expect(
      screen.getByTitle("Stage 2 is locked: Speed Controller (ESC) (Complete Stage 1 first)"),
    ).toBeDisabled();
    expect(
      screen.getByTitle("Stage 3 is locked: Video Transmitter (VTX) (Complete Stages 1 & 2 first)"),
    ).toBeDisabled();

    // Clicking locked Flight Controller button does NOT navigate to Stage 2
    await user.click(fcPartButton);
    expect(screen.queryByText("2A. Flight Controller")).not.toBeInTheDocument();

    // Advance to Stage 1 and select Frame, Motor, and Propeller to complete Stage 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));

    // Verify 3 / 11 parts are now selected and Stage 1 is complete
    expect(screen.getByText("3 / 11 Parts")).toBeInTheDocument();

    // Now Stage 2 is unlocked in the Selected Parts list
    const unlockedFcButton = screen.getByTitle("Go to Stage 2: Flight Controller selection");
    expect(unlockedFcButton).toBeEnabled();

    // Clicking on Flight Controller now successfully navigates to Stage 2
    await user.click(unlockedFcButton);
    expect(screen.getByText("2A. Flight Controller")).toBeInTheDocument();
  });

  it("displays product hover card when hovering over a configured component in the selected parts list", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Stage 1 and select Frame
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));
    await user.click(await screen.findByText("Master 5 V2"));

    // Find the Selected Parts list frame button
    const frameBtn = screen.getByTitle("Go to Stage 1: Frame selection");
    expect(frameBtn).toBeInTheDocument();

    // Find the hover trigger wrapper
    const hoverTriggers = screen.getAllByTestId("product-hover-trigger");
    expect(hoverTriggers.length).toBeGreaterThanOrEqual(1);

    // Hover over the frame item
    fireEvent.mouseEnter(hoverTriggers[0]);

    // Hover card appears with Frame details
    await waitFor(
      () => {
        expect(screen.getByTestId("product-hover-card")).toBeInTheDocument();
      },
      { timeout: 1500 },
    );

    const hoverCard = screen.getByTestId("product-hover-card");
    expect(within(hoverCard).getByText("Master 5 V2")).toBeInTheDocument();
    expect(within(hoverCard).getByRole("link", { name: /details/i })).toBeInTheDocument();

    // Mouse leave hides the hover card
    fireEvent.mouseLeave(hoverTriggers[0]);
    await waitFor(
      () => {
        expect(screen.queryByTestId("product-hover-card")).not.toBeInTheDocument();
      },
      { timeout: 1500 },
    );
  });

  describe("RX Antenna Guard (rxHasIntegratedAntenna & Section 2D)", () => {
    it("rxHasIntegratedAntenna correctly identifies integrated vs external receivers", () => {
      // Null / undefined check
      expect(rxHasIntegratedAntenna(null)).toBe(false);
      expect(rxHasIntegratedAntenna(undefined)).toBe(false);

      // External antenna receiver (e.g. Crossfire Nano RX)
      expect(rxHasIntegratedAntenna(mockReceiver1)).toBe(false);

      // Receiver with integrated ceramic antenna in antennaUuids
      expect(rxHasIntegratedAntenna(mockReceiverCeramic)).toBe(true);

      // Receiver with "ceramic" in description
      const customCeramicRx = {
        ...mockReceiver1,
        description: "Receiver with built-in ceramic antenna for micros",
      };
      expect(rxHasIntegratedAntenna(customCeramicRx as any)).toBe(true);

      // Receiver with "smd antenna" in name
      const customSmdRx = {
        ...mockReceiver1,
        name: "ELRS 2.4GHz SMD Antenna Nano",
      };
      expect(rxHasIntegratedAntenna(customSmdRx as any)).toBe(true);
    });

    it("locks 'No External Antenna (None)' unless active receiver has an integrated ceramic antenna and auto-clears when switching to an external-antenna receiver", async () => {
      const user = userEvent.setup();
      renderWizard();

      // Advance to Stage 1 and select Frame, Motor, Prop
      await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));
      await user.click(await screen.findByText("Master 5 V2"));
      await user.click(await screen.findByText("ECO II 2207"));
      await user.click(await screen.findByText("Hurricane 51433"));

      // Advance to Stage 2
      await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));

      // Select FC and ESC
      await user.click(screen.getByText("F405 V4 FC"));
      await user.click(screen.getByText("SpeedyBee 50A 4-in-1 ESC"));

      // 1. Initial State in Stage 2 before RX is selected:
      // "No External Antenna (None)" should be locked
      const noneAntCard = screen
        .getByText("No External Antenna (None)")
        .closest("div[class*='border']") as HTMLElement;
      expect(noneAntCard).toBeInTheDocument();
      expect(noneAntCard?.className).toContain("cursor-not-allowed");
      expect(within(noneAntCard).getByText(/Antenna Required/i)).toBeInTheDocument();

      // 2. Select an external-antenna receiver: Crossfire Nano RX
      await user.click(screen.getByText("Crossfire Nano RX"));

      // "No External Antenna (None)" remains locked and shows that Crossfire Nano requires an external antenna
      expect(noneAntCard?.className).toContain("cursor-not-allowed");
      expect(
        within(noneAntCard).getByText(/Crossfire Nano RX requires an external antenna/i),
      ).toBeInTheDocument();

      // Clicking it does NOT select none (guard blocks it)
      await user.click(noneAntCard);
      expect(noneAntCard?.className).not.toContain("ring-blue-500");

      // 3. Select an external antenna: RadioMaster T-Antenna
      await user.click(screen.getByText("RadioMaster T-Antenna 2.4GHz"));

      // 4. Switch to an integrated ceramic receiver: RadioMaster RP2
      await user.click(screen.getByText("RadioMaster RP2 2.4GHz Receiver"));

      // Now "No External Antenna (None)" is unlocked!
      expect(noneAntCard?.className).toContain("cursor-pointer");
      expect(noneAntCard?.className).not.toContain("cursor-not-allowed");
      expect(within(noneAntCard).getByText("Integrated SMD Ceramic")).toBeInTheDocument();
      expect(
        within(noneAntCard).getByText(/Uses RadioMaster RP2.*onboard ceramic antenna/i),
      ).toBeInTheDocument();

      // Clicking "No External Antenna (None)" successfully selects it!
      await user.click(noneAntCard);
      expect(noneAntCard?.className).toContain("ring-blue-500");

      // Selected parts list reflects integrated ceramic antenna
      expect(screen.getByText("None (Integrated ceramic antenna)")).toBeInTheDocument();

      // 5. Switch back to Crossfire Nano RX (which requires an external antenna)
      await user.click(screen.getByText("Crossfire Nano RX"));

      // Safety check: "No External Antenna" must automatically clear and lock again!
      expect(noneAntCard?.className).toContain("cursor-not-allowed");
      expect(noneAntCard?.className).not.toContain("ring-blue-500");
      expect(within(noneAntCard).getByText(/Antenna Required/i)).toBeInTheDocument();
      expect(screen.queryByText("None (Integrated ceramic antenna)")).not.toBeInTheDocument();
    });
  });

  it("scrolls smoothly to selected product when clicking Selected in the section header", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Stage 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

    // Mock scrollIntoView
    const scrollIntoViewMock = vi.fn();
    window.HTMLElement.prototype.scrollIntoView = scrollIntoViewMock;

    // Select Frame (Master 5 V2)
    const frameItem = await screen.findByText("Master 5 V2");
    await user.click(frameItem);

    // Verify "Scroll to selected frame" button appears in 1A header
    const selectedBtn = screen.getByTitle("Scroll to selected frame");
    expect(selectedBtn).toBeInTheDocument();

    // Click "Selected" button
    await user.click(selectedBtn);

    // scrollIntoView should have been called on the selected item element
    expect(scrollIntoViewMock).toHaveBeenCalledWith({ behavior: "smooth", block: "center" });
  });

  describe("getVtxAntennaCount", () => {
    it("returns 1 for null or undefined VTX", () => {
      expect(getVtxAntennaCount(null)).toBe(1);
      expect(getVtxAntennaCount(undefined)).toBe(1);
    });

    it("returns 1 for analog VTX without antenna configuration", () => {
      expect(getVtxAntennaCount({ name: "TBS Unify Pro32", protocol: "Analog" } as any)).toBe(1);
    });

    it("returns 1 for DJI O4 Lite VTX", () => {
      expect(getVtxAntennaCount({ name: "DJI O4 Lite VTX", protocol: "DJI O4 Lite" } as any)).toBe(
        1,
      );
      expect(getVtxAntennaCount({ name: "DJI O4 Wide VTX", protocol: "DJI O4 Lite" } as any)).toBe(
        1,
      );
    });

    it("returns 2 for DJI O4 Pro VTX", () => {
      expect(
        getVtxAntennaCount({ name: "DJI O4 Air Unit Pro", protocol: "DJI O4 Pro" } as any),
      ).toBe(2);
      expect(
        getVtxAntennaCount({ name: "TANQ 2 DJI O4 Pro VTX Mount", protocol: "DJI O4 Pro" } as any),
      ).toBe(2);
    });

    it("returns 2 for DJI O3 and Walksnail Moonlight", () => {
      expect(getVtxAntennaCount({ name: "DJI O3 Air Unit", protocol: "DJI O3" } as any)).toBe(2);
      expect(
        getVtxAntennaCount({
          name: "Walksnail Moonlight VTX",
          protocol: "Walksnail Avatar",
        } as any),
      ).toBe(2);
    });

    it("respects explicit antennaUuids length when present", () => {
      expect(
        getVtxAntennaCount({
          name: "Custom VTX",
          protocol: "Analog",
          antennaUuids: ["ant-1", "ant-2"],
        } as any),
      ).toBe(2);
      expect(
        getVtxAntennaCount({
          name: "Custom VTX",
          protocol: "Analog",
          antennaUuids: ["ant-1"],
        } as any),
      ).toBe(1);
    });
  });
});
