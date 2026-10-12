import { describe, it, expect, vi } from "vitest";
import { screen, waitFor, within, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import {
  BuildWizardPage,
  rxHasIntegratedAntenna,
  getVtxAntennaCount,
  getRxAntennaCount,
} from "../BuildWizardPage";
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

  it("renders with initial empty state: Step 0 active with baseline options", async () => {
    renderWizard();

    // Check title & banner
    const heading = await screen.findByRole("heading", { level: 1, name: "The Forge" });
    expect(heading).toBeInTheDocument();
    const headingContainer = heading.parentElement;
    expect(headingContainer?.querySelector("svg")).toBeInTheDocument();
    expect(headingContainer?.querySelector("svg")).toHaveClass("text-[#FF6D00]");
    expect(screen.queryByText(/Interactive Build Wizard/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/• (Stage|Step)/i)).not.toBeInTheDocument();
    expect(screen.queryByText("Step 0: Choose a Starting Baseline")).not.toBeInTheDocument();

    // Step 0 active, Step 1 unlocked (0/3), steps 2-4 disabled
    expect(screen.getByText("Template Selection")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /step 1/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /step 2/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /step 3/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /step 4/i })).toBeDisabled();

    // Step 0 baseline options
    expect(screen.getByText("Choose Starting Baseline")).toBeInTheDocument();
    expect(screen.queryByText("Start from Scratch")).not.toBeInTheDocument();
    // Initial dry weight is 0.00g across evaluator and selected parts list
    expect(screen.getAllByText("0.00g").length).toBeGreaterThanOrEqual(1);

    // Selected Parts section under Evaluator (4 optional parts default to None)
    expect(screen.getByText("Selected Parts")).toBeInTheDocument();
    expect(screen.getByText("4 / 11 Parts")).toBeInTheDocument();
  });

  it("allows searching for frames and filtering products in real time", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
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

  it("allows selecting Step 1 parts, dynamically updates motor count, and unlocks Step 2", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
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
      expect(screen.queryByText("0.00g")).not.toBeInTheDocument();
    });

    // Next button should be enabled
    const nextBtn = screen.getByRole("button", { name: /Next: Flight Electronics/i });
    expect(nextBtn).toBeEnabled();

    // Advance to Step 2
    await user.click(nextBtn);
    expect(await screen.findByText("2A. Flight Controller")).toBeInTheDocument();
  });

  it("handles conditional ESC and Receiver selection: standalone FC requires separate components, AIO FC allows integrated components with rich specs", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

    // Select Step 1 parts
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));

    // Go to Step 2
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

    // Integrated FC ESC card is automatically selected when AIO FC is chosen!
    expect(screen.getByText("BetaFPV Integrated 20A ESC")).toBeInTheDocument();
    expect(screen.getByText("20A")).toBeInTheDocument();
    expect(screen.getByText(/4x Motors • BLHeli_S/i)).toBeInTheDocument();
    expect(
      screen.getByText("Use Integrated FC ESC").closest(".border-emerald-500"),
    ).toBeInTheDocument();

    // Clicking the selected integrated ESC deselects it
    await user.click(screen.getByText("Use Integrated FC ESC"));
    expect(screen.getByText("Use Integrated FC ESC").closest(".border-emerald-500")).toBeNull();

    // Integrated FC Receiver card is automatically selected when AIO FC is chosen!
    expect(screen.getByText("BetaFPV Integrated ELRS 2.4GHz RX")).toBeInTheDocument();
    expect(screen.getAllByText("ExpressLRS").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/2.4 GHz • Telemetry/i)).toBeInTheDocument();
    expect(
      screen.getByText("Use Integrated FC Receiver").closest(".border-emerald-500"),
    ).toBeInTheDocument();

    // Clicking the selected integrated RX deselects it
    await user.click(screen.getByText("Use Integrated FC Receiver"));
    expect(
      screen.getByText("Use Integrated FC Receiver").closest(".border-emerald-500"),
    ).toBeNull();
  });

  it("handles Step 2 RX antenna & GPS and Step 3 Video optional components with 'Set All to None'", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

    // Complete Step 1
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));

    // Verify Step 2 sections: 2A FC, 2B ESC, 2C RX, 2D RX Antenna, 2E GPS
    expect(await screen.findByText("2A. Flight Controller")).toBeInTheDocument();
    expect(screen.getByText("2B. Electronic Speed Controller (ESC)")).toBeInTheDocument();
    expect(screen.getByText("2C. Radio Control Receiver")).toBeInTheDocument();
    expect(screen.getByText("2D. Radio Receiver Antenna(s)")).toBeInTheDocument();
    expect(screen.getByText("2E. GPS Receiver & Compass")).toBeInTheDocument();

    // Select Step 2 components
    await user.click(screen.getByText("F405 V4 FC"));
    await user.click(screen.getByText("SpeedyBee 50A 4-in-1 ESC"));
    await user.click(screen.getByText("Crossfire Nano RX"));

    // Select compatible 915MHz RX Antenna for Crossfire Nano RX
    const rxAntCard = screen.getByText("TBS Immortal T V2 915MHz");
    await user.click(rxAntCard);
    expect(screen.getByText("Antenna Quantity:")).toBeInTheDocument();
    expect(screen.getByText("1x")).toBeInTheDocument();

    // Select GPS in Step 2
    const gpsCard = await screen.findByText(/M8Q-5883/i);
    await user.click(gpsCard);

    // Advance to Step 3 (Video)
    const nextBtn = screen.getByRole("button", { name: /Next: Video/i });
    expect(nextBtn).toBeEnabled();
    await user.click(nextBtn);

    // Verify Step 3 Video sections: 3A VTX, 3B Camera, 3C Video Transmitter Antenna(s)
    expect(
      await screen.findByText(/All video components in Step 3 are optional/i),
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

    // Verify explicit None selections in Step 3
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
    expect(screen.getByText("+10.00g")).toBeInTheDocument();

    // Click GoPro preset (+133g)
    const goproBtn = screen.getByRole("button", { name: /GoPro \(\+133g\)/i });
    await user.click(goproBtn);
    expect(screen.getByText("+133.00g")).toBeInTheDocument();

    // Click Bare preset (0g)
    const bareBtn = screen.getByRole("button", { name: /Bare \(0g\)/i });
    await user.click(bareBtn);
    expect(screen.getByText("+0.00g")).toBeInTheDocument();
  });

  it("allows selecting a template in Step 0, reviewing BOM in Step 4, and saving the build to PostgreSQL", async () => {
    const user = userEvent.setup();
    renderWizard();

    // In Step 0, select "Bando Basher 5 inch" template
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);

    // Template should be active and Steps 1-4 unlocked
    expect(screen.getByText("Template Active")).toBeInTheDocument();

    // All steps 0 to 4 in the step bar should become marked as Done
    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const stepTabs = within(stepNav).getAllByRole("button");
    expect(stepTabs).toHaveLength(5);
    for (const tab of stepTabs) {
      expect(tab).toHaveTextContent("Done");
    }

    // Navigate to Step 4 (Review & Save)
    const step4Tab = within(stepNav).getByRole("button", { name: /Step 4/i });
    expect(step4Tab).toBeEnabled();
    await user.click(step4Tab);

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

    // Select template in Step 0
    const templateCards = await screen.findAllByText("Bando Basher 5 inch");
    await user.click(templateCards[0]);

    // Navigate to Step 4 (Review & Save)
    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const step4Tab = within(stepNav).getByRole("button", { name: /Step 4/i });
    await user.click(step4Tab);

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

  it("resets all wizard selections and returns to Step 0 on Reset button click", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Select template in Step 0
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);
    expect(screen.getByText("Template Active")).toBeInTheDocument();

    // Advance to Step 1
    await user.click(screen.getByRole("button", { name: /Next: Airframe & Propulsion/i }));
    expect(screen.getByText("1A. Frame Chassis")).toBeInTheDocument();

    // Click Reset button in the Start Over card
    expect(screen.getByText("Start Over")).toBeInTheDocument();
    const resetBtn = screen.getByRole("button", { name: /Reset/i });
    await user.click(resetBtn);

    // Should return to Step 0 baseline selection
    expect(screen.getByText("Choose Starting Baseline")).toBeInTheDocument();
    expect(screen.getAllByText("0.00g").length).toBeGreaterThanOrEqual(1);
  });

  it("displays dynamic CEL filter when physics engine is active and allows toggling compatible parts", async () => {
    vi.spyOn(wasmEngine, "getIsReady").mockReturnValue(true);
    vi.spyOn(wasmEngine, "generateCelFilter").mockImplementation((target: string) => {
      if (target === "propellers") return "diameter_mm <= 130.0";
      return "";
    });

    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

    // Select Frame (Master 5 V2 - 130mm max prop size)
    await user.click(await screen.findByText("Master 5 V2"));

    // Check that Compatible only has dotted underline and shows hover card with copy button on hover
    const compatibleLabel = await screen.findByText(/compatible only/i);
    expect(compatibleLabel).toBeInTheDocument();
    expect(compatibleLabel.className).toContain("decoration-dotted");

    // Hovering over "Compatible only" reveals the hover card
    await user.hover(compatibleLabel);
    const hoverCard = await screen.findByTestId("compatibility-hover-card");
    expect(hoverCard).toBeInTheDocument();
    expect(hoverCard.querySelector("pre")).toHaveTextContent("diameter_mm <= 130.0");
    expect(screen.getByTestId("copy-filter-button")).toBeInTheDocument();

    // Toggle off Compatible Only
    const compatibleCheckbox = screen.getByRole("checkbox", { name: /compatible only/i });
    expect(compatibleCheckbox).toBeChecked();
    await user.click(compatibleCheckbox);
    expect(compatibleCheckbox).not.toBeChecked();
  });

  it("renders the selected parts list under the build evaluator and updates when components are selected", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Verify Selected Parts card under Evaluator
    expect(screen.getByText("Selected Parts")).toBeInTheDocument();
    expect(screen.getByText("4 / 11 Parts")).toBeInTheDocument();
    expect(screen.getByText("BOM Total Dry Weight:")).toBeInTheDocument();

    // Verify Step 1 items are unlocked initially
    expect(screen.getByTitle("Go to Step 1: Frame selection")).toBeEnabled();
    expect(screen.getByTitle("Go to Step 1: Motor selection")).toBeEnabled();
    expect(screen.getByTitle("Go to Step 1: Propeller selection")).toBeEnabled();

    // Verify Step 2 & 3 items are locked initially with disabled buttons
    const fcPartButton = screen.getByTitle(
      "Step 2 is locked: Flight Controller (Complete Step 1 first)",
    );
    expect(fcPartButton).toBeDisabled();
    expect(
      screen.getByTitle("Step 2 is locked: Speed Controller (ESC) (Complete Step 1 first)"),
    ).toBeDisabled();
    expect(
      screen.getByTitle("Step 3 is locked: Video Transmitter (VTX) (Complete Steps 1 & 2 first)"),
    ).toBeDisabled();

    // Clicking locked Flight Controller button does NOT navigate to Step 2
    await user.click(fcPartButton);
    expect(screen.queryByText("2A. Flight Controller")).not.toBeInTheDocument();

    // Advance to Step 1 and select Frame, Motor, and Propeller to complete Step 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));

    // Verify 7 / 11 parts are now selected (3 propulsion + 4 default None) and Step 1 is complete
    expect(screen.getByText("7 / 11 Parts")).toBeInTheDocument();

    // Now Step 2 is unlocked in the Selected Parts list
    const unlockedFcButton = screen.getByTitle("Go to Step 2: Flight Controller selection");
    expect(unlockedFcButton).toBeEnabled();

    // Clicking on Flight Controller now successfully navigates to Step 2
    await user.click(unlockedFcButton);
    expect(screen.getByText("2A. Flight Controller")).toBeInTheDocument();
  });

  it("displays product hover card when hovering over a configured component in the selected parts list", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1 and select Frame
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));
    await user.click(await screen.findByText("Master 5 V2"));

    // Find the Selected Parts list frame button
    const frameBtn = screen.getByTitle("Go to Step 1: Frame selection");
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

    it("requires radio receiver antenna selection to complete Step 2 and does not provide a None option", async () => {
      const user = userEvent.setup();
      renderWizard();

      // Advance to Step 1 and select Frame, Motor, Prop
      await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));
      await user.click(await screen.findByText("Master 5 V2"));
      await user.click(await screen.findByText("ECO II 2207"));
      await user.click(await screen.findByText("Hurricane 51433"));

      // Advance to Step 2
      await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));

      // Select FC, ESC, and RX
      await user.click(screen.getByText("F405 V4 FC"));
      await user.click(screen.getByText("SpeedyBee 50A 4-in-1 ESC"));
      await user.click(screen.getByText("Crossfire Nano RX"));

      // 1. Verify that "No External Antenna (None)" is NOT present
      expect(screen.queryByText("No External Antenna (None)")).not.toBeInTheDocument();

      // 2. Verify that Section 2D displays "Required"
      const rxAntSection = screen
        .getByText("2D. Radio Receiver Antenna(s)")
        .closest("div[id='section-rx-ant']") as HTMLElement;
      expect(within(rxAntSection).getByText("Required")).toBeInTheDocument();

      // Next step button should be disabled because RX antenna is not selected yet
      const nextBtn = screen.getByRole("button", { name: /Next: Video/i });
      expect(nextBtn).toBeDisabled();

      // 3. Select a compatible 915MHz RX antenna
      await user.click(screen.getByText("TBS Immortal T V2 915MHz"));

      // Section 2D now shows "Selected"
      expect(within(rxAntSection).getByText("Selected")).toBeInTheDocument();

      // Next step button is now enabled
      expect(nextBtn).not.toBeDisabled();
    });
  });

  it("scrolls smoothly to selected product when clicking Selected in the section header", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
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

  it("keeps 'Compatible Only' checkbox visible when deselected, and renders incompatible products with desaturation and warning badge", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

    // Select Frame (Master 5 V2 - maxPropSizeMm: 130)
    await user.click(await screen.findByText("Master 5 V2"));

    // By default, "Compatible only" is checked in 1C Propellers
    const compatCheckbox = (await screen.findByLabelText(/compatible only/i)) as HTMLInputElement;
    expect(compatCheckbox).toBeInTheDocument();
    expect(compatCheckbox.checked).toBe(true);

    // Only compatible propeller (Hurricane 51433) is visible; incompatible 7-inch (HQProp DP 7X4X3) is not visible
    expect(screen.getByText("Hurricane 51433")).toBeInTheDocument();
    expect(screen.queryByText("HQProp DP 7X4X3")).not.toBeInTheDocument();

    // Uncheck "Compatible Only"
    await user.click(compatCheckbox);

    // Checkbox and banner must NOT disappear!
    expect(compatCheckbox).toBeInTheDocument();
    expect(compatCheckbox.checked).toBe(false);

    // Now incompatible propeller (HQProp DP 7X4X3) is visible
    const incompatibleProp = await screen.findByText("HQProp DP 7X4X3");
    expect(incompatibleProp).toBeInTheDocument();

    // Verify incompatible styling and badge
    const card = incompatibleProp.closest(".cursor-pointer");
    expect(card?.className).toContain("opacity-60");
    expect(card?.className).toContain("saturate-50");
    expect(
      within(card as HTMLElement).getByText(/Incompatible \(7.0" > 5.1"\)/i),
    ).toBeInTheDocument();

    // Re-check "Compatible Only"
    await user.click(compatCheckbox);
    expect(compatCheckbox.checked).toBe(true);
    expect(screen.queryByText("HQProp DP 7X4X3")).not.toBeInTheDocument();
  });

  it("navigates to step and scrolls to selected product when clicking on a selected part in the sidebar", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

    // Mock scrollIntoView
    const scrollIntoViewMock = vi.fn();
    window.HTMLElement.prototype.scrollIntoView = scrollIntoViewMock;

    // Select Frame, Motor, Prop
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));

    // Advance to Step 2
    await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));
    expect(await screen.findByText("2A. Flight Controller")).toBeInTheDocument();

    // Find "Frame" item in "Selected Parts" sidebar and click it
    const sidebarFrameBtn = screen.getByTitle("Go to Step 1: Frame selection");
    expect(sidebarFrameBtn).toBeInTheDocument();

    scrollIntoViewMock.mockClear();
    await user.click(sidebarFrameBtn);

    // Verify it transitioned back to Step 1
    expect(screen.getByText("1A. Frame Chassis")).toBeInTheDocument();

    // Wait for the smooth scroll to be triggered on the selected element
    await waitFor(() => {
      expect(scrollIntoViewMock).toHaveBeenCalledWith({ behavior: "smooth", block: "center" });
    });
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

  describe("getRxAntennaCount", () => {
    it("returns 1 for null or undefined RX", () => {
      expect(getRxAntennaCount(null)).toBe(1);
      expect(getRxAntennaCount(undefined)).toBe(1);
    });

    it("returns 1 for standard single-antenna RX", () => {
      expect(
        getRxAntennaCount({
          name: "Crossfire Nano RX",
          protocol: "CRSF",
        } as any),
      ).toBe(1);
      expect(
        getRxAntennaCount({
          name: "BetaFPV ELRS 2.4G",
          protocol: "ExpressLRS",
        } as any),
      ).toBe(1);
    });

    it("returns 2 for diversity RX by name", () => {
      expect(
        getRxAntennaCount({
          name: "RadioMaster RP3 Diversity Receiver",
          protocol: "ExpressLRS",
        } as any),
      ).toBe(2);
      expect(
        getRxAntennaCount({
          name: "BetaFPV SuperD Diversity RX",
          protocol: "ExpressLRS",
        } as any),
      ).toBe(2);
    });

    it("returns 2 for Gemini / dual protocol RX", () => {
      expect(
        getRxAntennaCount({
          name: "BetaFPV SuperG",
          protocol: "Gemini",
        } as any),
      ).toBe(2);
      expect(
        getRxAntennaCount({
          name: "TBS Crossfire Diversity RX",
          protocol: "CRSF Dual",
        } as any),
      ).toBe(2);
    });

    it("respects explicit antennaUuids length when present", () => {
      expect(
        getRxAntennaCount({
          name: "Custom RX",
          antennaUuids: ["ant-1", "ant-2"],
        } as any),
      ).toBe(2);
      expect(
        getRxAntennaCount({
          name: "Custom RX",
          antennaUuids: ["ant-1"],
        } as any),
      ).toBe(1);
    });
  });

  it("allows deselecting any component by clicking on the selected card", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));

    // 1A. Select Frame then click again to deselect
    const frameCard = await screen.findByText("Master 5 V2");
    await user.click(frameCard);
    expect(frameCard.closest(".border-blue-500")).toBeInTheDocument();
    await user.click(frameCard);
    expect(frameCard.closest(".border-blue-500")).toBeNull();

    // Re-select frame for downstream steps
    await user.click(frameCard);

    // 1B. Select Motor then click again to deselect
    const motorCard = await screen.findByText("ECO II 2207");
    await user.click(motorCard);
    expect(motorCard.closest(".border-blue-500")).toBeInTheDocument();
    await user.click(motorCard);
    expect(motorCard.closest(".border-blue-500")).toBeNull();

    // Re-select motor
    await user.click(motorCard);

    // 1C. Select Propeller then click again to deselect
    const propCard = await screen.findByText("Hurricane 51433");
    await user.click(propCard);
    expect(propCard.closest(".border-blue-500")).toBeInTheDocument();
    await user.click(propCard);
    expect(propCard.closest(".border-blue-500")).toBeNull();

    // Re-select propeller
    await user.click(propCard);

    // Advance to Step 2
    await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));

    // 2A. Select standalone FC then click again to deselect
    const fcCard = await screen.findByText("F405 V4 FC");
    await user.click(fcCard);
    expect(fcCard.closest(".border-blue-500")).toBeInTheDocument();
    await user.click(fcCard);
    expect(fcCard.closest(".border-blue-500")).toBeNull();

    // Re-select FC
    await user.click(fcCard);

    // 2B. Select Standalone ESC then click again to deselect
    const escCard = await screen.findByText("SpeedyBee 50A 4-in-1 ESC");
    await user.click(escCard);
    expect(escCard.closest(".border-blue-500")).toBeInTheDocument();
    await user.click(escCard);
    expect(escCard.closest(".border-blue-500")).toBeNull();

    // 2C. Select RX then click again to deselect
    const rxCard = await screen.findByText("Crossfire Nano RX");
    await user.click(rxCard);
    expect(rxCard.closest(".border-blue-500")).toBeInTheDocument();
    await user.click(rxCard);
    expect(rxCard.closest(".border-blue-500")).toBeNull();

    // Re-select RX
    await user.click(rxCard);

    // 2D. Select compatible 915MHz antenna then click again to deselect
    const rxAntCard = await screen.findByText("TBS Immortal T V2 915MHz");
    await user.click(rxAntCard);
    expect(rxAntCard.closest(".border-blue-500")).toBeInTheDocument();
    await user.click(rxAntCard);
    expect(rxAntCard.closest(".border-blue-500")).toBeNull();

    // 2E. Select GPS then click again to deselect
    const gpsCard = await screen.findByText(/M8Q-5883/i);
    await user.click(gpsCard);
    expect(gpsCard.closest(".border-blue-500")).toBeInTheDocument();
    await user.click(gpsCard);
    expect(gpsCard.closest(".border-blue-500")).toBeNull();
  });

  it("filters RX antenna options by frequency matching the selected receiver", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1 & complete baseline
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));

    // Select FC and Crossfire (915 MHz) RX
    await user.click(await screen.findByText("F405 V4 FC"));
    await user.click(await screen.findByText("Crossfire Nano RX"));

    // In 915 MHz mode with Compatible Only active:
    // TBS Immortal T V2 915MHz should be visible
    expect(screen.getByText("TBS Immortal T V2 915MHz")).toBeInTheDocument();
    // RadioMaster T-Antenna 2.4GHz should NOT be visible
    expect(screen.queryByText("RadioMaster T-Antenna 2.4GHz")).not.toBeInTheDocument();
  });

  it("formats motor stator, prop diameter with mm/inch, VTX protocol chip, and defaults optional parts to None", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Advance to Step 1
    await user.click(await screen.findByRole("button", { name: /Next: Airframe & Propulsion/i }));
    await user.click(await screen.findByText("Master 5 V2"));

    // Verify motor card stator size format (2207 from formatStatorSize)
    expect(screen.getByText("2207")).toBeInTheDocument();

    // Verify propeller card diameter format: diameter in mm with inches in parentheses
    expect(screen.getByText(/129\.5mm \(5\.1"\)/)).toBeInTheDocument();

    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(await screen.findByText("Hurricane 51433"));

    // Advance to Step 2
    await user.click(screen.getByRole("button", { name: /Next: Flight Electronics/i }));

    // Verify GPS defaults to "No GPS (None)" selected
    const noGpsCard = screen.getByText("No GPS (None)").closest(".cursor-pointer");
    expect(noGpsCard).toHaveClass("border-blue-500");

    // Select a GPS product
    const gpsCard = await screen.findByText(/M8Q-5883/i);
    await user.click(gpsCard);
    expect(gpsCard.closest(".border-blue-500")).toBeInTheDocument();
    expect(noGpsCard).not.toHaveClass("border-blue-500");

    // Deselect the GPS product by clicking it again -> reverts to "No GPS (None)"
    await user.click(gpsCard);
    expect(gpsCard.closest(".border-blue-500")).toBeNull();
    expect(noGpsCard).toHaveClass("border-blue-500");

    // Select FC, ESC, RX, and RX antenna to complete Step 2
    await user.click(await screen.findByText("F405 V4 FC"));
    await user.click(await screen.findByText("SpeedyBee 50A 4-in-1 ESC"));
    await user.click(await screen.findByText("Crossfire Nano RX"));
    await user.click(await screen.findByText("TBS Immortal T V2 915MHz"));

    // Advance to Step 3 (Video)
    await user.click(screen.getByRole("button", { name: /Next: Video/i }));

    // Verify optional components in Step 3 default to None
    const noVtxAntCard = screen.getByText("No Antenna (None)").closest(".cursor-pointer");
    expect(noVtxAntCard).toHaveClass("border-blue-500");
    const noCamCard = screen.getByText("No Camera (None)").closest(".cursor-pointer");
    expect(noCamCard).toHaveClass("border-blue-500");

    // Verify VTX product card chip shows protocol ("SmartAudio"), not transmit power ("1000mW")
    expect(screen.getByText("SmartAudio")).toBeInTheDocument();
    expect(screen.queryByText("1000mW")).not.toBeInTheDocument();
  });
});
