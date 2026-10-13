import { describe, it, expect, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
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
  mockFrame1,
  mockReceiver1,
  mockReceiverCeramic,
} from "../../test/mocks/fixtures";
import { checkVtxAntennaCompatibility } from "../../components/forge/ForgeContext";
import { create } from "@bufbuild/protobuf";
import { VideoTransmitterSchema } from "../../gen/quadsmith/api/video_transmitter_pb";
import { AntennaSchema } from "../../gen/quadsmith/api/antenna_pb";
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

  async function confirmScratchBaseline(user: ReturnType<typeof userEvent.setup>) {
    const scratchCard = await screen.findByText(/No Template \(Start from Scratch\)/i);
    await user.click(scratchCard);
  }

  async function advanceToStep1(user: ReturnType<typeof userEvent.setup>) {
    await confirmScratchBaseline(user);
    const nextBtn = await screen.findByRole("button", { name: /Next: Frame/i });
    await waitFor(() => expect(nextBtn).toBeEnabled());
    await user.click(nextBtn);
    await screen.findByText("1. Frame Chassis");
  }

  it("renders with initial empty state: Step 0 active, Step 1 locked until baseline confirmation", async () => {
    renderWizard();

    // Check title & banner
    const heading = await screen.findByRole("heading", { level: 1, name: "The Forge" });
    expect(heading).toBeInTheDocument();
    const headingContainer = heading.parentElement;
    expect(headingContainer?.querySelector("svg")).toBeInTheDocument();
    expect(headingContainer?.querySelector("svg")).toHaveClass("text-[#FF6D00]");

    // Step 0 active in step bar, Step 1 locked (disabled) pending confirmation
    expect(screen.getByRole("button", { name: /^step 0:/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /^step 1:/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /^step 2:/i })).toBeDisabled();

    // Step 0 baseline options
    expect(screen.getByText("Choose Starting Baseline")).toBeInTheDocument();
    expect(screen.getByText(/No Template \(Start from Scratch\)/i)).toBeInTheDocument();

    // Next: Frame button is disabled pending explicit confirmation
    const nextBtn = screen.getByRole("button", { name: /Next: Frame/i });
    expect(nextBtn).toBeDisabled();

    // Initial dry weight is 0.00g
    expect(screen.getAllByText("0.00g").length).toBeGreaterThanOrEqual(1);

    // Selected Parts section under Evaluator
    expect(screen.getByText("Selected Parts")).toBeInTheDocument();
  });

  it("requires positive confirmation for Step 0 ('No template') before unlocking Step 1 and enabling Next", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Initially Step 1 is locked and Next: Frame is disabled
    expect(screen.getByRole("button", { name: /^step 1:/i })).toBeDisabled();
    const nextBtn = screen.getByRole("button", { name: /Next: Frame/i });
    expect(nextBtn).toBeDisabled();

    // Click "No template (Start from scratch)"
    await confirmScratchBaseline(user);

    // Now Step 0 shows confirmed and Step 1 is unlocked
    expect(screen.getByText("Confirmed")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^step 1:/i })).toBeEnabled();
    expect(nextBtn).toBeEnabled();

    // Advance to Step 1 (Frame Chassis)
    await user.click(nextBtn);
    expect(await screen.findByText("1. Frame Chassis")).toBeInTheDocument();
  });

  it("allows searching for frames and filtering products in real time", async () => {
    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);

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

  it("progresses sequentially through dedicated component steps (Frame -> FC -> RX -> RX Antenna -> ESC -> Motors -> Props)", async () => {
    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);

    // Step 1: Select Frame
    expect(await screen.findByText("1. Frame Chassis")).toBeInTheDocument();
    await user.click(await screen.findByText("Master 5 V2"));
    const nextToFc = screen.getByRole("button", { name: /Next: Flight Controller/i });
    expect(nextToFc).toBeEnabled();
    await user.click(nextToFc);

    // Step 2: Select Flight Controller
    expect(await screen.findByText("2. Flight Controller")).toBeInTheDocument();
    await user.click(await screen.findByText("F405 V4 FC"));
    const nextToRx = screen.getByRole("button", { name: /Next: Radio Receiver/i });
    expect(nextToRx).toBeEnabled();
    await user.click(nextToRx);

    // Step 3: Select Receiver
    expect(await screen.findByText("3. Radio Control Receiver")).toBeInTheDocument();
    await user.click(await screen.findByText("Crossfire Nano RX"));
    const nextToRxAnt = screen.getByRole("button", { name: /Next: RX Antennas/i });
    expect(nextToRxAnt).toBeEnabled();
    await user.click(nextToRxAnt);

    // Step 4: Select Receiver Antenna
    expect(await screen.findByText("4. Radio Receiver Antenna(s)")).toBeInTheDocument();
    await user.click(await screen.findByText("TBS Immortal T V2 915MHz"));
    const nextToEsc = screen.getByRole("button", { name: /Next: Speed Controller/i });
    expect(nextToEsc).toBeEnabled();
    await user.click(nextToEsc);

    // Step 5: Select ESC
    expect(await screen.findByText("5. Electronic Speed Controller (ESC)")).toBeInTheDocument();
    await user.click(await screen.findByText("SpeedyBee 50A 4-in-1 ESC"));
    const nextToMotors = screen.getByRole("button", { name: /Next: Motors/i });
    expect(nextToMotors).toBeEnabled();
    await user.click(nextToMotors);

    // Step 6: Select Motors
    expect(await screen.findByText("6. Motors")).toBeInTheDocument();
    await user.click(await screen.findByText("ECO II 2207"));
    const nextToProps = screen.getByRole("button", { name: /Next: Propellers/i });
    expect(nextToProps).toBeEnabled();
    await user.click(nextToProps);

    // Step 7: Select Propeller
    expect(await screen.findByText("7. Propellers")).toBeInTheDocument();
    await user.click(await screen.findByText("Hurricane 51433"));

    // Dry weight should now be > 0
    await waitFor(() => {
      expect(screen.queryByText("0.00g")).not.toBeInTheDocument();
    });

    const nextToCam = screen.getByRole("button", { name: /Next: FPV Camera/i });
    expect(nextToCam).toBeEnabled();
  });

  it("handles conditional ESC and Receiver selection: standalone FC requires separate components, AIO FC allows integrated components with rich specs", async () => {
    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Controller/i }));

    // Select standalone FC (F405 V4)
    await user.click(await screen.findByText("F405 V4 FC"));
    await user.click(screen.getByRole("button", { name: /Next: Radio Receiver/i }));

    // Step 3 (Receiver) with standalone FC: Integrated RX is disabled
    expect(
      screen.getByText("Use Integrated FC Receiver").closest(".cursor-not-allowed"),
    ).toBeInTheDocument();

    // Navigate back to Step 2 (Flight Controller) and select AIO FC
    await user.click(screen.getByRole("button", { name: /^step 2:/i }));
    await user.click(await screen.findByText("Integrated FC AIO"));

    // Go to Step 3 (Receiver): Integrated RX is auto-selected!
    await user.click(screen.getByRole("button", { name: /^step 3:/i }));
    expect(
      screen.getAllByText(/BetaFPV Integrated ELRS 2\.4GHz RX/i).length,
    ).toBeGreaterThanOrEqual(1);
    expect(
      screen.getByText("Use Integrated FC Receiver").closest(".border-blue-500"),
    ).toBeInTheDocument();

    // Clicking the selected integrated RX keeps it selected (deselection is disabled)
    await user.click(screen.getByText("Use Integrated FC Receiver"));
    expect(
      screen.getByText("Use Integrated FC Receiver").closest(".border-blue-500"),
    ).toBeInTheDocument();
  });

  it("automatically skips Receiver Antenna step when receiver has integrated SMD/ceramic antenna", async () => {
    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Controller/i }));
    await user.click(await screen.findByText("F405 V4 FC"));
    await user.click(screen.getByRole("button", { name: /Next: Radio Receiver/i }));

    // Select RP2 (Receiver with built-in SMD ceramic antenna)
    await user.click(await screen.findByText("RadioMaster RP2 2.4GHz Receiver"));

    // Next button in Step 3 should skip Step 4 (RX Antennas) and go directly to Step 5 (ESC)!
    const nextBtn = screen.getByRole("button", { name: /Next: ESC \(Antenna Integrated\)/i });
    expect(nextBtn).toBeInTheDocument();
    await user.click(nextBtn);

    // Verify we landed on Step 5 (ESC)
    expect(await screen.findByText("5. Electronic Speed Controller (ESC)")).toBeInTheDocument();
  });

  it("handles optional Camera, VTX, VTX Antenna, and GPS steps with Omitted / None options", async () => {
    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Controller/i }));
    await user.click(await screen.findByText("F405 V4 FC"));
    await user.click(screen.getByRole("button", { name: /Next: Radio Receiver/i }));
    await user.click(await screen.findByText("Crossfire Nano RX"));
    await user.click(screen.getByRole("button", { name: /Next: RX Antennas/i }));
    await user.click(await screen.findByText("TBS Immortal T V2 915MHz"));
    await user.click(screen.getByRole("button", { name: /Next: Speed Controller/i }));
    await user.click(await screen.findByText("SpeedyBee 50A 4-in-1 ESC"));
    await user.click(screen.getByRole("button", { name: /Next: Motors/i }));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(screen.getByRole("button", { name: /Next: Propellers/i }));
    await user.click(await screen.findByText("Hurricane 51433"));
    await user.click(screen.getByRole("button", { name: /Next: FPV Camera/i }));

    // Step 8: FPV Camera
    expect(await screen.findByText("8. FPV Camera")).toBeInTheDocument();
    expect(screen.getByText("No Camera (Omitted)")).toBeInTheDocument();

    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const step9Tab = within(stepNav).getByRole("button", { name: /^step 9:/i });
    const step10Tab = within(stepNav).getByRole("button", { name: /^step 10:/i });
    const step11Tab = within(stepNav).getByRole("button", { name: /^step 11:/i });
    const step12Tab = within(stepNav).getByRole("button", { name: /^step 12:/i });

    // Omit Camera: Step 9 & 10 are locked to None and disabled in stepNav
    await user.click(screen.getByText("No Camera (Omitted)"));
    expect(step9Tab).toBeDisabled();
    expect(step10Tab).toBeDisabled();
    expect(step11Tab).toBeEnabled();
    // Step 12 is also unlocked because GPS is None by default
    expect(step12Tab).toBeEnabled();

    // Verify step bar subtitles show Locked (Omitted) for Step 9 & 10 and None (Omitted) for Step 11
    expect(within(step9Tab).getByText("Locked (Omitted)")).toBeInTheDocument();
    expect(within(step10Tab).getByText("Locked (Omitted)")).toBeInTheDocument();
    expect(within(step11Tab).getByText("None (Omitted)")).toBeInTheDocument();

    // Next button in Step 8 skips to Step 11 (GPS)
    const nextBtn = screen.getByRole("button", { name: /Next: GPS \(Video Skipped\)/i });
    expect(nextBtn).toBeInTheDocument();
    await user.click(nextBtn);

    // Verify skipped directly to Step 11 (GPS)
    expect(await screen.findByText("11. GPS Receiver & Compass")).toBeInTheDocument();

    // Verify GPS is selected as "Omit GPS (Freestyle / Racing)" by default
    expect(screen.getAllByText("None (Omitted)").length).toBeGreaterThanOrEqual(1);

    // Next button from Step 11 goes to Step 12 (Review & Save) without requiring explicit interaction
    const nextToReviewBtn = screen.getByRole("button", { name: /Next: Review/i });
    expect(nextToReviewBtn).toBeEnabled();

    // Back button from Step 11 skips Step 10 and Step 9, returning directly to Step 8 (Camera)
    const backBtn = screen.getByRole("button", { name: /Back: Camera/i });
    expect(backBtn).toBeInTheDocument();
    await user.click(backBtn);
    expect(await screen.findByText("8. FPV Camera")).toBeInTheDocument();

    // Selecting a camera in Step 8 unlocks Step 9 (VTX) and Step 10 (VTX Antenna)
    const ratelCam = await screen.findByText("Ratel 2");
    await user.click(ratelCam);
    expect(step9Tab).toBeEnabled();
    expect(
      screen.getByRole("button", { name: /Next: Video Transmitter \(VTX\)/i }),
    ).toBeInTheDocument();
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

  it("allows selecting a template in Step 0, reviewing BOM in Step 12, and saving the build to PostgreSQL", async () => {
    const user = userEvent.setup();
    renderWizard();

    // In Step 0, select "Bando Basher 5 inch" template
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);

    // Template should be active and all steps unlocked
    expect(screen.getByText("Template Active")).toBeInTheDocument();

    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const stepTabs = within(stepNav).getAllByRole("button");
    expect(stepTabs).toHaveLength(13);

    // Navigate to Step 12 (Review & Save)
    const step12Tab = within(stepNav).getByRole("button", { name: /^step 12:/i });
    expect(step12Tab).toBeEnabled();
    await user.click(step12Tab);

    // Check BOM items populated from template
    expect(await screen.findByText(/Bill of Materials \(BOM\)/i)).toBeInTheDocument();
    expect(screen.getAllByText(/ECO II 2207/i).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText(/F405 V4 FC/i).length).toBeGreaterThanOrEqual(1);

    // Calculated Build Metrics section should NOT be present (metrics are in live evaluator)
    expect(screen.queryByText(/Calculated Build Metrics/i)).toBeNull();

    // Verify Public/Private switch toggle is present and defaults to Public
    const visibilitySwitch = screen.getByRole("switch", { name: "Build Visibility" });
    expect(visibilitySwitch).toBeInTheDocument();
    expect(visibilitySwitch).toHaveAttribute("aria-checked", "true");
    expect(screen.getByText("Public Build")).toBeInTheDocument();

    // Toggle switch to Private
    await user.click(visibilitySwitch);
    expect(visibilitySwitch).toHaveAttribute("aria-checked", "false");
    expect(screen.getByText("Private Build")).toBeInTheDocument();

    // Click Save Build
    const saveBtn = screen.getByRole("button", { name: /Save & Create Build/i });
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

    // Navigate to Step 12 (Review & Save)
    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const step12Tab = within(stepNav).getByRole("button", { name: /^step 12:/i });
    await user.click(step12Tab);

    // Change build name to one that generates -copy1
    const nameInput = screen.getByLabelText("Build Name");
    await user.clear(nameInput);
    await user.type(nameInput, "Bando Basher 5 inch copy1");

    // Click Save Build
    const saveBtn = screen.getByRole("button", { name: /Save & Create Build/i });
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
    await user.click(screen.getByRole("button", { name: /Next: Frame/i }));
    expect(screen.getByText("1. Frame Chassis")).toBeInTheDocument();

    // Click Reset button in the top banner
    const resetBtn = screen.getByRole("button", { name: /Reset Build/i });
    await user.click(resetBtn);

    // Should return to Step 0 baseline selection
    expect(screen.getByText("Choose Starting Baseline")).toBeInTheDocument();
    expect(screen.getAllByText("0.00g").length).toBeGreaterThanOrEqual(1);
  });

  it("displays dynamic CEL filter when physics engine is active and allows toggling compatible parts", async () => {
    vi.spyOn(wasmEngine, "getIsReady").mockReturnValue(true);
    vi.spyOn(wasmEngine, "checkCompatibility").mockReturnValue({ messages: [] } as any);
    vi.spyOn(wasmEngine, "generateCelFilter").mockImplementation((target: string) => {
      if (target === "propellers") return "diameter_mm <= 130.0";
      return "";
    });

    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);

    // Frame step should NOT have a Compatible only checkbox
    expect(screen.queryByRole("checkbox", { name: /compatible only/i })).toBeNull();

    // Select Frame (Master 5 V2 - 130mm max prop size)
    await user.click(await screen.findByText("Master 5 V2"));

    // Advance to Step 2 (Flight Controller)
    const nextBtn = screen.getByRole("button", { name: /next/i });
    await user.click(nextBtn);

    // Check that Compatible only toggle works in Step 2
    const compatibleCheckbox = screen.getByRole("checkbox", { name: /compatible only/i });
    expect(compatibleCheckbox).toBeChecked();
    await user.click(compatibleCheckbox);
    expect(compatibleCheckbox).not.toBeChecked();
  });

  it("renders the selected frame in the frame list when 'Bando Basher 5 inch' template is selected", async () => {
    const user = userEvent.setup();
    renderWizard();

    // In Step 0, select "Bando Basher 5 inch" template
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);

    // Advance to Step 1
    await user.click(screen.getByRole("button", { name: /Next: Frame/i }));

    // Verify Step 1 Frame step has no "Compatible only" toggle
    expect(screen.queryByRole("checkbox", { name: /compatible only/i })).toBeNull();

    // Verify the frame from the template is displayed in the list and selected
    const selectedFrameName = mockFrame1.name;
    const frameCards = await screen.findAllByText(selectedFrameName);
    expect(frameCards.length).toBeGreaterThan(0);

    // Verify that the selected frame element exists
    const selectedItem = document.getElementById("selected-frame-item");
    expect(selectedItem).not.toBeNull();
  });

  it("renders the selected parts list under the build evaluator and updates when components are selected", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Verify Selected Parts card under Evaluator
    expect(screen.getByText("Selected Parts")).toBeInTheDocument();
    expect(screen.getByText("0 / 11 Parts")).toBeInTheDocument();
    expect(screen.getByText((_, el) => el?.textContent === "Dry: 0.0g")).toBeInTheDocument();

    // Confirm Step 0 and select Frame in Step 1
    await advanceToStep1(user);
    await user.click(await screen.findByText("Master 5 V2"));

    // Selected parts count and dry weight increment
    expect(screen.getByText("1 / 11 Parts")).toBeInTheDocument();
    expect(screen.getByText((_, el) => el?.textContent === "Dry: 145.0g")).toBeInTheDocument();
    expect(screen.getAllByText("145.0g").length).toBeGreaterThanOrEqual(2);
  });

  it("scrolls smoothly to selected product when clicking Selected in the section header", async () => {
    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);

    // Mock scrollIntoView
    const scrollIntoViewMock = vi.fn();
    window.HTMLElement.prototype.scrollIntoView = scrollIntoViewMock;

    // Select Frame (Master 5 V2)
    const frameItem = await screen.findByText("Master 5 V2");
    await user.click(frameItem);

    // Verify "Scroll to selected" button appears
    const selectedBtn = screen.getByTitle("Scroll to selected frame");
    expect(selectedBtn).toBeInTheDocument();

    // Click "Selected" button
    await user.click(selectedBtn);
    expect(scrollIntoViewMock).toHaveBeenCalledWith({ behavior: "smooth", block: "center" });
  });

  it("keeps 'Compatible Only' checkbox visible when deselected, and renders incompatible products with desaturation and warning badge", async () => {
    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Controller/i }));
    await user.click(await screen.findByText("F405 V4 FC"));
    await user.click(screen.getByRole("button", { name: /Next: Radio Receiver/i }));
    await user.click(await screen.findByText("Crossfire Nano RX"));
    await user.click(screen.getByRole("button", { name: /Next: RX Antennas/i }));
    await user.click(await screen.findByText("TBS Immortal T V2 915MHz"));
    await user.click(screen.getByRole("button", { name: /Next: Speed Controller/i }));
    await user.click(await screen.findByText("SpeedyBee 50A 4-in-1 ESC"));
    await user.click(screen.getByRole("button", { name: /Next: Motors/i }));
    await user.click(await screen.findByText("ECO II 2207"));
    await user.click(screen.getByRole("button", { name: /Next: Propellers/i }));

    // By default, "Compatible only" is checked in Step 7 Propellers
    const compatCheckbox = (await screen.findByLabelText(/compatible only/i)) as HTMLInputElement;
    expect(compatCheckbox).toBeInTheDocument();
    expect(compatCheckbox.checked).toBe(true);

    // Only compatible propeller (Hurricane 51433) is visible; incompatible 7-inch (HQProp DP 7X4X3) is not visible
    expect(screen.getByText("Hurricane 51433")).toBeInTheDocument();
    expect(screen.queryByText("HQProp DP 7X4X3")).not.toBeInTheDocument();

    // Uncheck "Compatible Only"
    await user.click(compatCheckbox);
    expect(compatCheckbox.checked).toBe(false);

    // Now incompatible propeller (HQProp DP 7X4X3) is visible
    const incompatibleProp = await screen.findByText("HQProp DP 7X4X3");
    expect(incompatibleProp).toBeInTheDocument();

    // Re-check "Compatible Only"
    await user.click(compatCheckbox);
    expect(compatCheckbox.checked).toBe(true);
    expect(screen.queryByText("HQProp DP 7X4X3")).not.toBeInTheDocument();
  });

  it("navigates to step and scrolls to selected product when clicking on a selected part in the sidebar", async () => {
    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Controller/i }));
    expect(await screen.findByText("2. Flight Controller")).toBeInTheDocument();

    // Click "Frame" in the Selected Parts sidebar list
    const sidebarFramePart = screen
      .getAllByText("Frame")
      .find((el) => el.closest("[role='button']"))
      ?.closest("[role='button']") as HTMLElement;
    expect(sidebarFramePart).toBeInTheDocument();
    await user.click(sidebarFramePart);

    // Verify transition back to Step 1
    expect(screen.getByText("1. Frame Chassis")).toBeInTheDocument();
  });

  describe("RX Antenna Guard (rxHasIntegratedAntenna & Frequency Matching)", () => {
    it("rxHasIntegratedAntenna correctly identifies integrated vs external receivers", () => {
      expect(rxHasIntegratedAntenna(null)).toBe(false);
      expect(rxHasIntegratedAntenna(undefined)).toBe(false);
      expect(rxHasIntegratedAntenna(mockReceiver1)).toBe(false);
      expect(rxHasIntegratedAntenna(mockReceiverCeramic)).toBe(true);

      const customCeramicRx = {
        ...mockReceiver1,
        description: "Receiver with built-in ceramic antenna for micros",
      };
      expect(rxHasIntegratedAntenna(customCeramicRx as any)).toBe(true);

      const customSmdRx = {
        ...mockReceiver1,
        name: "ELRS 2.4GHz SMD Antenna Nano",
      };
      expect(rxHasIntegratedAntenna(customSmdRx as any)).toBe(true);
    });

    it("filters RX antenna options by frequency matching the selected receiver", async () => {
      const user = userEvent.setup();
      renderWizard();

      await advanceToStep1(user);
      await user.click(await screen.findByText("Master 5 V2"));
      await user.click(screen.getByRole("button", { name: /Next: Flight Controller/i }));
      await user.click(await screen.findByText("F405 V4 FC"));
      await user.click(screen.getByRole("button", { name: /Next: Radio Receiver/i }));

      // Select Crossfire Nano RX (915 MHz)
      await user.click(await screen.findByText("Crossfire Nano RX"));
      await user.click(screen.getByRole("button", { name: /Next: RX Antennas/i }));

      // Step 4 (Receiver Antennas):
      // TBS Immortal T V2 915MHz should be visible
      expect(screen.getByText("TBS Immortal T V2 915MHz")).toBeInTheDocument();
      // 2.4GHz antenna should NOT be visible when compatible-only is active
      expect(screen.queryByText("RadioMaster T-Antenna 2.4GHz")).not.toBeInTheDocument();
    });
  });

  describe("VTX Antenna Count & protocol derivation", () => {
    it("returns 1 for null or undefined VTX", () => {
      expect(getVtxAntennaCount(null)).toBe(1);
      expect(getVtxAntennaCount(undefined)).toBe(1);
    });

    it("returns 1 for analog VTX without antenna configuration", () => {
      expect(getVtxAntennaCount({ name: "TBS Unify", protocol: "SmartAudio" } as any)).toBe(1);
    });

    it("returns 1 for DJI O4 Lite VTX", () => {
      expect(
        getVtxAntennaCount({ name: "DJI O4 Lite Air Unit", protocol: "DJI O4 Lite" } as any),
      ).toBe(1);
    });

    it("returns 2 for DJI O4 Pro VTX", () => {
      expect(
        getVtxAntennaCount({ name: "DJI O4 Pro Air Unit", protocol: "DJI O4 Pro" } as any),
      ).toBe(2);
    });

    it("returns 2 for DJI O3 and Walksnail Moonlight", () => {
      expect(getVtxAntennaCount({ name: "DJI O3 Air Unit", protocol: "DJI O3" } as any)).toBe(2);
      expect(
        getVtxAntennaCount({
          name: "Walksnail Moonlight Kit",
          protocol: "Walksnail Avatar",
        } as any),
      ).toBe(2);
    });

    it("respects explicit antennaUuids length when present", () => {
      expect(
        getVtxAntennaCount({
          name: "Custom VTX",
          protocol: "Analog",
          antennaUuids: ["uuid-1", "uuid-2"],
        } as any),
      ).toBe(2);
    });
  });

  describe("RX Antenna Count derivation", () => {
    it("returns 1 for null or undefined RX", () => {
      expect(getRxAntennaCount(null)).toBe(1);
      expect(getRxAntennaCount(undefined)).toBe(1);
    });

    it("returns 1 for standard single-antenna RX", () => {
      expect(getRxAntennaCount({ name: "Crossfire Nano", protocol: "CRSF" } as any)).toBe(1);
    });

    it("returns 2 for diversity RX by name", () => {
      expect(getRxAntennaCount({ name: "Crossfire Diversity RX", protocol: "CRSF" } as any)).toBe(
        2,
      );
      expect(
        getRxAntennaCount({ name: "True Diversity ELRS", protocol: "ExpressLRS" } as any),
      ).toBe(2);
    });

    it("returns 2 for Gemini / dual protocol RX", () => {
      expect(getRxAntennaCount({ name: "SuperG Gemini RX", protocol: "ExpressLRS" } as any)).toBe(
        2,
      );
      expect(getRxAntennaCount({ name: "Dual Antenna RX", protocol: "ExpressLRS" } as any)).toBe(2);
    });

    it("respects explicit antennaUuids length when present", () => {
      expect(
        getRxAntennaCount({
          name: "Standard RX",
          protocol: "ExpressLRS",
          antennaUuids: ["ant-1", "ant-2"],
        } as any),
      ).toBe(2);
    });
  });

  it("prevents deselecting components when clicking on the already selected card", async () => {
    const user = userEvent.setup();
    renderWizard();

    await advanceToStep1(user);

    // Select Frame then click again: selection remains intact
    const frameCard = await screen.findByText("Master 5 V2");
    await user.click(frameCard);
    expect(frameCard.closest(".border-blue-500")).toBeInTheDocument();
    await user.click(frameCard);
    expect(frameCard.closest(".border-blue-500")).toBeInTheDocument();
    expect(screen.getByText("1 / 11 Parts")).toBeInTheDocument();
  });

  it("shows evaluation paused notice when no battery is selected and clears it when battery is chosen", async () => {
    const user = userEvent.setup();
    renderWizard();

    // In Step 0 scratch baseline, no battery is selected initially
    expect(screen.getByText(/Evaluation Paused/i)).toBeInTheDocument();
    expect(
      screen.getByText(/Select a test battery to calculate thrust-to-weight/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/Select Battery to Calculate/i)).toBeInTheDocument();

    // Select a battery
    const batteryCombobox = screen.getByRole("combobox", { name: /select test battery/i });
    await user.click(batteryCombobox);
    const batteryOption = await screen.findByText(/Black Series 1500mAh 6S/i);
    await user.click(batteryOption);

    // Notice disappears
    expect(screen.queryByText(/Evaluation Paused/i)).toBeNull();
  });

  it("formats motor stator and prop diameter with mm/inch correctly", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Select template to load parts immediately
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);

    // Navigate to Step 6 (Motors)
    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    await user.click(within(stepNav).getByRole("button", { name: /^step 6:/i }));

    // Verify motor card stator size format (2207)
    expect(screen.getByText("2207")).toBeInTheDocument();

    // Navigate to Step 7 (Propellers)
    await user.click(within(stepNav).getByRole("button", { name: /^step 7:/i }));

    // Verify propeller card diameter format: mm with inches in parentheses
    expect(screen.getByText(/129\.5mm \(5\.1"\)/)).toBeInTheDocument();
  });

  it("provides blank first option in battery combobox and opens BatteryPickerModal on Browse click", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Verify battery combobox is rendered
    const batteryCombobox = screen.getByRole("combobox", { name: /select test battery/i });
    expect(batteryCombobox).toBeInTheDocument();

    // Click combobox to open dropdown
    await user.click(batteryCombobox);
    expect(await screen.findByRole("listbox", { name: "Batteries" })).toBeInTheDocument();
    expect(screen.getByText("None (No battery selected)")).toBeInTheDocument();

    // Browse button next to battery combobox
    const browseBtn = screen.getByRole("button", { name: /browse all compatible batteries/i });
    expect(browseBtn).toBeInTheDocument();

    // Clicking Browse opens BatteryPickerModal
    await user.click(browseBtn);
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText(/Choose Compatible Battery/i)).toBeInTheDocument();
  });

  it("disables subsequent steps when an incompatible component is selected, keeping downstream selections intact", async () => {
    vi.spyOn(wasmEngine, "getIsReady").mockReturnValue(true);
    vi.spyOn(wasmEngine, "checkCompatibility").mockImplementation((req: any) => {
      const comp = req.components;
      const messages: any[] = [];
      if (comp?.propeller && comp?.frame && comp.propeller.diameterMm > comp.frame.maxPropSizeMm) {
        messages.push({
          checkerName: "PropellerFrameChecker",
          message: "Propeller diameter exceeds frame maximum",
        });
      }
      return { messages } as any;
    });

    const user = userEvent.setup();
    renderWizard();

    // Select template to populate all components
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);

    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const step7Tab = within(stepNav).getByRole("button", { name: /^step 7:/i });
    const step8Tab = within(stepNav).getByRole("button", { name: /^step 8:/i });
    const step12Tab = within(stepNav).getByRole("button", { name: /^step 12:/i });

    // Await template application and verification that steps are enabled
    await waitFor(() => expect(step7Tab).toBeEnabled());
    expect(step8Tab).toBeEnabled();
    expect(step12Tab).toBeEnabled();

    // Go to Step 7 (Propellers)
    await user.click(step7Tab);
    expect(await screen.findByText("7. Propellers")).toBeInTheDocument();

    // Uncheck "Compatible only" to show incompatible 7-inch props on 5-inch frame
    const compatCheckbox = await screen.findByLabelText(/compatible only/i);
    await user.click(compatCheckbox);

    // Select incompatible 7-inch propeller (HQProp DP 7X4X3, 177.8mm exceeds 130mm max prop size)
    const incompatibleProp = await screen.findByText("HQProp DP 7X4X3");
    await user.click(incompatibleProp);

    // Incompatibility banner / message should appear on Step 7
    expect(await screen.findByText(/Incompatibility Detected/i)).toBeInTheDocument();

    // Subsequent steps (Step 8 and Step 12) are now LOCKED / disabled
    expect(step8Tab).toBeDisabled();
    expect(step12Tab).toBeDisabled();

    // Downstream selections are NOT wiped (e.g. Caddx Ratel 2 camera from template is still selected in sidebar)
    expect(screen.getAllByText(/Ratel 2/i).length).toBeGreaterThanOrEqual(1);

    // Selecting a compatible propeller again resolves the conflict and unlocks subsequent steps
    const compatibleProp = await screen.findByText("Hurricane 51433");
    await user.click(compatibleProp);

    await waitFor(() => expect(screen.queryByText(/Incompatibility Detected/i)).toBeNull());
    expect(step8Tab).toBeEnabled();
    expect(step12Tab).toBeEnabled();
  });

  it("flags battery voltage mismatch in step bar and highlights incompatible step with alert icon", async () => {
    vi.spyOn(wasmEngine, "getIsReady").mockReturnValue(true);
    vi.spyOn(wasmEngine, "checkCompatibility").mockReturnValue({ messages: [] } as any);

    const user = userEvent.setup();
    renderWizard();

    // Select template to populate components (SpeedyBee 50A ESC with minVoltage: 11.1V)
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);

    // Await template application
    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const step5Tab = within(stepNav).getByRole("button", { name: /^step 5:/i });
    await waitFor(() => expect(step5Tab).toBeEnabled());

    // Select 1S battery with maxVoltage 4.35V (incompatible with 11.1V min ESC voltage)
    const batteryCombobox = screen.getByRole("combobox", { name: /select test battery/i });
    await user.click(batteryCombobox);
    const incompatibleBatteryOption = await screen.findByText(/Tattu 450mAh 1S/i);
    await user.click(incompatibleBatteryOption);

    // Step 5 (ESC) should have battery conflict alert in WizardStepBar
    await waitFor(() => {
      expect(step5Tab.querySelector("svg.text-amber-500")).toBeTruthy();
    });
  });

  it("does not auto-select analog integrated VTX when digital camera is selected", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Load template baseline so all steps are complete
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);

    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const step2Tab = within(stepNav).getByRole("button", { name: /^step 2:/i });
    const step8Tab = within(stepNav).getByRole("button", { name: /^step 8:/i });
    const step9Tab = within(stepNav).getByRole("button", { name: /^step 9:/i });
    await waitFor(() => expect(step2Tab).toBeEnabled());

    // Navigate to Step 2 (FC) and select FC with integrated analog VTX
    await user.click(step2Tab);
    await user.click(await screen.findByText("BetaFPV F4 1S AIO VTX"));

    // Navigate to Step 8 (Camera) and select digital camera (DJI O3 Air Unit Camera)
    await user.click(step8Tab);
    await user.click(await screen.findByText("DJI O3 Air Unit Camera"));

    // Navigate to Step 9 (VTX)
    await user.click(step9Tab);
    expect(await screen.findByText("9. Video Transmitter (VTX)")).toBeInTheDocument();

    // Verify integrated analog VTX was NOT auto-selected
    expect(screen.queryByText("Use FC Integrated VTX")).toBeNull();
  });

  describe("VTX Antenna Compatibility Checks (checkVtxAntennaCompatibility)", () => {
    it("returns compatible for matching frequency band and connector", () => {
      const vtx = create(VideoTransmitterSchema, {
        name: "Analog 5.8G VTX",
        frequencyBandMhz: 5800,
        antennaConnectors: ["SMA"],
      });
      const ant = create(AntennaSchema, {
        name: "Lollipop 4 SMA",
        frequencyBandMhz: 5800,
        connector: "SMA",
      });
      const result = checkVtxAntennaCompatibility(vtx, ant);
      expect(result.compatible).toBe(true);
    });

    it("flags frequency band mismatch between VTX and antenna", () => {
      const vtx = create(VideoTransmitterSchema, {
        name: "SpeedyBee TX800 5.8GHz VTX",
        frequencyBandMhz: 5800,
        antennaConnectors: ["SMA"],
      });
      const ant = create(AntennaSchema, {
        name: "ELRS 2.4GHz Antenna",
        frequencyBandMhz: 2400,
        connector: "SMA",
      });
      const result = checkVtxAntennaCompatibility(vtx, ant);
      expect(result.compatible).toBe(false);
      expect(result.reason).toContain("Frequency mismatch");
    });

    it("flags connector type mismatch between VTX and antenna", () => {
      const vtx = create(VideoTransmitterSchema, {
        name: "DJI O3 Air Unit",
        frequencyBandMhz: 5800,
        antennaConnectors: ["Dual U.FL", "U.FL"],
      });
      const ant = create(AntennaSchema, {
        name: "Foxeer Lollipop SMA",
        frequencyBandMhz: 5800,
        connector: "SMA",
      });
      const result = checkVtxAntennaCompatibility(vtx, ant);
      expect(result.compatible).toBe(false);
      expect(result.reason).toContain("Connector mismatch");
    });
  });

  it("enforces frequency band and connector type compatibility for VTX antennas in Step 10", async () => {
    vi.spyOn(wasmEngine, "getIsReady").mockReturnValue(true);
    vi.spyOn(wasmEngine, "generateCelFilter").mockReturnValue("");
    vi.spyOn(wasmEngine, "checkCompatibility").mockImplementation((req: any) => {
      const messages: any[] = [];
      const comp = req?.components;
      if (comp?.videoTransmitter && comp?.antennas?.length > 0) {
        const vtx = comp.videoTransmitter;
        const vtxAnt = comp.antennas[comp.antennas.length - 1];
        const res = checkVtxAntennaCompatibility(vtx, vtxAnt);
        if (!res.compatible) {
          messages.push({
            checkerName: "VideoTransmitterAntennaFrequencyChecker",
            message: res.reason,
          });
        }
      }
      return { messages } as any;
    });
    const user = userEvent.setup();
    renderWizard();

    // Load template baseline so all steps are complete
    const templateCard = await screen.findByText("Bando Basher 5 inch");
    await user.click(templateCard);

    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const step10Tab = within(stepNav).getByRole("button", { name: /^step 10:/i });
    const step11Tab = within(stepNav).getByRole("button", { name: /^step 11:/i });
    const step12Tab = within(stepNav).getByRole("button", { name: /^step 12:/i });
    await waitFor(() => expect(step10Tab).toBeEnabled());

    // Navigate to Step 10 (VTX Antenna)
    await user.click(step10Tab);
    expect(await screen.findByText("10. VTX Antenna(s)")).toBeInTheDocument();

    // Uncheck "Compatible only" to show all antennas
    const compatCheckbox = screen.getByLabelText(/compatible only/i);
    await user.click(compatCheckbox);

    // Select incompatible 2.4GHz antenna
    const incompatibleAnt = await screen.findByText("RadioMaster T-Antenna 2.4GHz");
    await user.click(incompatibleAnt);

    // Incompatibility banner should appear on Step 10
    expect(await screen.findByText(/Incompatibility Detected/i)).toBeInTheDocument();
    expect(screen.getAllByText(/Frequency mismatch/i).length).toBeGreaterThanOrEqual(1);

    // Downstream steps should be locked
    expect(step11Tab).toBeDisabled();
    expect(step12Tab).toBeDisabled();

    // Re-select compatible antenna (Lollipop 4 RHCP)
    const compatibleAnt = await screen.findByText("Lollipop 4 RHCP");
    await user.click(compatibleAnt);

    // Incompatibility banner should be gone and downstream steps unlocked
    await waitFor(() => expect(screen.queryByText(/Incompatibility Detected/i)).toBeNull());
    expect(step11Tab).toBeEnabled();
    expect(step12Tab).toBeEnabled();
  });

  it("defaults optional components (GPS) to None and auto-selects integrated components when FC is chosen", async () => {
    const user = userEvent.setup();
    renderWizard();

    // Confirm scratch baseline
    await advanceToStep1(user);

    // Select Frame in Step 1
    await user.click(await screen.findByText("Master 5 V2"));
    await user.click(screen.getByRole("button", { name: /Next: Flight Controller/i }));

    // Select FC with integrated components (Integrated FC AIO)
    await user.click(await screen.findByText("Integrated FC AIO"));

    // GPS in step bar displays None (Omitted) by default
    const stepNav = screen.getByRole("navigation", { name: "Build Steps" });
    const step11Tab = within(stepNav).getByRole("button", { name: /^step 11:/i });
    expect(within(step11Tab).getByText("None (Omitted)")).toBeInTheDocument();

    // Navigate to Step 3 (Radio Receiver)
    await user.click(screen.getByRole("button", { name: /Next: Radio Receiver/i }));
    expect(screen.getByText("Use Integrated FC Receiver")).toBeInTheDocument();

    // Next goes to Step 4 (RX Antennas)
    await user.click(screen.getByRole("button", { name: /Next: RX Antennas/i }));
    expect(await screen.findByText("4. Radio Receiver Antenna(s)")).toBeInTheDocument();

    // Select antenna and proceed to Step 5 (ESC)
    await user.click(await screen.findByText("RadioMaster T-Antenna 2.4GHz"));
    await user.click(screen.getByRole("button", { name: /Next: Speed Controller/i }));

    // Step 5 (ESC): Integrated ESC is active
    expect(await screen.findByText("5. Electronic Speed Controller (ESC)")).toBeInTheDocument();
    expect(screen.getByText("Use Integrated FC ESC")).toBeInTheDocument();
  });
});
