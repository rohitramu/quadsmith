import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { BuildProfilePage } from "../BuildProfilePage";
import { Route, Routes } from "react-router-dom";

describe("BuildProfilePage Component", () => {
  function renderBuildProfile(buildId: string = "bando-basher-5-inch") {
    return renderWithProviders(
      <Routes>
        <Route path="/builds/:buildId" element={<BuildProfilePage />} />
      </Routes>,
      { route: `/builds/${buildId}` },
    );
  }

  it("renders breadcrumbs, build name, handle, and description", async () => {
    renderBuildProfile();

    expect(
      await screen.findByRole("heading", { level: 1, name: "Bando Basher 5 inch" }),
    ).toBeInTheDocument();
    expect(screen.getByText("@bando-basher-5-inch")).toBeInTheDocument();
    expect(
      screen.getByText(/durable 5-inch freestyle quadcopter built to withstand concrete hits/i),
    ).toBeInTheDocument();

    const breadcrumbs = screen.getByRole("navigation", { name: "Breadcrumb" });
    expect(breadcrumbs).toBeInTheDocument();
  });

  it("renders build evaluation metrics dashboard and compatibility badge", async () => {
    renderBuildProfile();

    expect(
      await screen.findByRole("heading", {
        level: 2,
        name: /build evaluation & performance/i,
      }),
    ).toBeInTheDocument();

    // Check AUW, TWR, Max Accel, Top Speed, Hover Throttle, Flight Time metrics
    expect(await screen.findByText("All-Up Weight")).toBeInTheDocument();
    expect(await screen.findByText("566")).toBeInTheDocument(); // 565.5 rounded to 566

    expect(screen.getByText("Thrust / Weight")).toBeInTheDocument();
    expect(screen.getByText("6.4")).toBeInTheDocument(); // 3645 / 565.5 ~ 6.4
    expect(screen.getByText("Freestyle Acro")).toBeInTheDocument();

    expect(screen.getByText("Max Acceleration")).toBeInTheDocument();
    expect(screen.getByText("53.4")).toBeInTheDocument();
    expect(screen.getByText(/5\.4 G vertical punchout/i)).toBeInTheDocument();

    expect(screen.getByText("Top Speed")).toBeInTheDocument();
    expect(screen.getByText("173")).toBeInTheDocument(); // 172.5 rounded to 173 km/h
    expect(screen.getByText(/107 mph terminal/i)).toBeInTheDocument();

    expect(screen.getByText("Hover Throttle")).toBeInTheDocument();
    expect(screen.getByText(/prop RPM/i)).toBeInTheDocument();
    expect(screen.getByText("Est. Flight Time")).toBeInTheDocument();
    expect(screen.getByText("3.8 – 7.2")).toBeInTheDocument();
    expect(screen.getByText(/varies with throttle management/i)).toBeInTheDocument();

    // All evaluated components are fully compatible and flight-ready
    expect(
      screen.getByText(/all evaluated components are fully compatible and flight-ready/i),
    ).toBeInTheDocument();
  });

  it("renders Bill of Materials (BOM) with hardware components and product links", async () => {
    renderBuildProfile();

    expect(
      await screen.findByRole("heading", {
        level: 2,
        name: /bill of materials \(hardware components\)/i,
      }),
    ).toBeInTheDocument();

    // Frame
    expect(await screen.findByText("Master 5 V2")).toBeInTheDocument();
    // Motor
    expect(await screen.findByText("ECO II 2207")).toBeInTheDocument();
    // Primary Battery is decoupled from BOM
    expect(screen.queryByText("Primary Battery")).not.toBeInTheDocument();
    // FC
    expect(await screen.findByText("F405 V4 FC")).toBeInTheDocument();
    // ESC
    expect(await screen.findByText("SpeedyBee 50A 4-in-1 ESC")).toBeInTheDocument();
    // Propeller
    expect(await screen.findByText("Hurricane 51433")).toBeInTheDocument();
    // Camera
    expect(await screen.findByText("Ratel 2")).toBeInTheDocument();
    // VTX
    expect(await screen.findByText("Unify Pro32 Nano")).toBeInTheDocument();
    // Receiver
    expect(await screen.findByText("Crossfire Nano RX")).toBeInTheDocument();
    // Antenna
    expect(await screen.findByText("Lollipop 4 RHCP")).toBeInTheDocument();
    // GPS
    expect(await screen.findByText("M8Q-5883 GPS & Compass")).toBeInTheDocument();
  });

  it("renders runtime battery selector with compatible batteries and allows modal exploration", async () => {
    const { user } = renderBuildProfile();

    // Verify battery combobox is present in runtime parameters
    const batterySelect = await screen.findByRole("combobox", { name: /select battery/i });
    expect(batterySelect).toBeInTheDocument();
    await waitFor(() => {
      expect(batterySelect).toHaveValue("cnhl-black-series-1500-6s");
    });

    // Click Browse button to open BatteryPickerModal
    const browseBtn = screen.getByRole("button", { name: /browse all compatible batteries/i });
    expect(browseBtn).toBeInTheDocument();
    await user.click(browseBtn);

    // Modal should be open with title and electrical limits
    expect(screen.getByText("Choose Compatible Battery")).toBeInTheDocument();
    expect(screen.getByText(/14.8v – 25.2v/i)).toBeInTheDocument();
  });

  it("allows interactive payload weight simulation with text box input and presets", async () => {
    const { user } = renderBuildProfile();

    expect(await screen.findByText("Payload")).toBeInTheDocument();

    const payloadInput = screen.getByRole("textbox", {
      name: /payload weight in grams/i,
    });
    expect(payloadInput).toBeInTheDocument();
    expect(payloadInput).toHaveValue("0");

    // Type a custom payload weight into the text box (e.g. 50g)
    await user.clear(payloadInput);
    await user.type(payloadInput, "50");
    expect(payloadInput).toHaveValue("50");

    // AUW should update: 565.5 + 50 = 615.5 -> 616g
    expect(await screen.findByText("616")).toBeInTheDocument();
    expect(screen.getByText("+50g")).toBeInTheDocument();

    // Click GoPro preset (+133g) to verify preset buttons update text box
    const goProButton = screen.getByRole("button", { name: "GoPro (+133g)" });
    await user.click(goProButton);

    expect(payloadInput).toHaveValue("133");
    // AUW should update to include payload: 565.5 + 133 = 698.5 -> 699g
    expect(await screen.findByText("699")).toBeInTheDocument();
    expect(screen.getByText("+133g")).toBeInTheDocument();
    expect(screen.getByText(/includes \+133g payload/i)).toBeInTheDocument();

    // Test + button to add 10 grams (133 + 10 = 143g)
    const plusButton = screen.getByRole("button", { name: /add 10 grams/i });
    await user.click(plusButton);
    expect(payloadInput).toHaveValue("143");
    expect(screen.getByText("+143g")).toBeInTheDocument();

    // Test - button to remove 10 grams (143 - 10 = 133g)
    const minusButton = screen.getByRole("button", { name: /remove 10 grams/i });
    await user.click(minusButton);
    expect(payloadInput).toHaveValue("133");
    expect(screen.getByText("+133g")).toBeInTheDocument();

    // Reset to Bare (0g) and verify - button is disabled
    const bareButton = screen.getByRole("button", { name: "Bare (0g)" });
    await user.click(bareButton);
    expect(payloadInput).toHaveValue("0");
    expect(minusButton).toBeDisabled();

    // Verify metric cards remain continuously mounted and present without layout collapse
    expect(screen.getByText("All-Up Weight")).toBeInTheDocument();
    expect(screen.getByText("Thrust / Weight")).toBeInTheDocument();
    expect(screen.getByText("Max Acceleration")).toBeInTheDocument();
    expect(screen.getByText("Top Speed")).toBeInTheDocument();
    expect(screen.getByText("Hover Throttle")).toBeInTheDocument();
    expect(screen.getByText("Est. Flight Time")).toBeInTheDocument();
  });

  it("renders reference documentation links", async () => {
    renderBuildProfile();

    expect(
      await screen.findByRole("heading", {
        level: 2,
        name: /reference links & documentation/i,
      }),
    ).toBeInTheDocument();

    expect(screen.getByText("https://github.com/tbs-trappy/source_one")).toBeInTheDocument();
  });

  it("displays not found error when build does not exist", async () => {
    renderBuildProfile("unknown-build-id");

    expect(
      await screen.findByRole("heading", { level: 1, name: "Build Not Found" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/could not find a build profile for "unknown-build-id"/i),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /return to builds feed/i })).toHaveAttribute(
      "href",
      "/",
    );
  });

  it("renders build primary display image and media showcase with images and video embeds", async () => {
    renderBuildProfile();

    expect(
      await screen.findByRole("heading", {
        level: 2,
        name: /build media & video showcase/i,
      }),
    ).toBeInTheDocument();

    expect(screen.getByAltText("Primary Display")).toBeInTheDocument();
    expect(screen.getByText("Bando Basher 5 inch Freestyle Frame")).toBeInTheDocument();
    expect(screen.getByText("Bando Basher Flight & Durability Test")).toBeInTheDocument();
    expect(screen.getByTitle("Bando Basher Flight & Durability Test")).toBeInTheDocument();
  });
});
