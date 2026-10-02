# Quadsmith Compatibility Checker Architecture

## 1. Overview
The compatibility checker evaluates an arbitrary set of components and produces a set of human-readable messages indicating physical, electrical, or software issues.

**Output Format**
The overall output is an array of evaluation result objects, grouping messages by the specific subset of components that generated them. This provides the UI with exact context and numerical telemetry to render visualizations (like gauges, sliders, or match lists).

- **`components`**: The specific subset of components evaluated together (e.g., the Motor and the ESC).
- **`checkerName`**: The name of the rule/checker that ran (e.g., `MotorEscElectricalChecker`).
- **`metrics`**: A strictly typed object containing the raw telemetry for this specific checker. Because the frontend explicitly knows about each checker's metrics, it maps these keys to bespoke UI components (e.g., `estimatedCurrentDraw: { value: 45, max: 40, unit: "A" }` or `fcMountMatch: { expected: ["20x20"], actual: ["30.5x30.5"] }`).
- **`messages`**: An array of issues found by this checker for these components.
  - **`severityLevel`**: Integer representing priority/severity (e.g., `0` for highest severity).
  - **`severityName`**: Human-readable enum string (e.g., `"DEFINITE_INCOMPATIBILITY"`, `"HEURISTIC_INCOMPATIBILITY"`, `"PARTIAL_COMPATIBILITY"`, `"INFORMATIONAL"`).
  - **`message`**: A clear, human-readable explanation of the issue.
  - **`resolution`**: An actionable suggestion to fix the problem.

**Save Behavior**
We prioritize user freedom. If a build has **`DEFINITE_INCOMPATIBILITY`** messages, the system will prominently display them, but will **allow the user to save the build anyway**. This accounts for custom modifications or missing data specs that we might not know about.

---

## 2. API & Data Loading
**Single Source of Truth**
The compatibility logic lives entirely in the backend to ensure consistency.

**API Operation**
The API exposes a standalone endpoint (e.g., `POST /api/compatibility/check`) that accepts an arbitrary array of component IDs. It does not require a saved `Build` entity—users can send any random assortment of components to see how they interact. 
1. **Batch Fetch**: The backend makes a single optimized query to fetch all specifications for the provided component IDs, alongside any explicit overrides from the `IncompatibilityIssue` table.
2. **Synchronous Evaluation**: The loaded data is passed into an array of in-memory TypeScript rules.
3. **Response**: The API returns an array of evaluated messages.

---

## 3. Codebase Structure
The engine will expose a single, flexible method that accepts an **arbitrary number of components** (from a single pair up to an entire build).

1. **Dynamic Instantiation:** Internally, the engine analyzes the provided list of components and dynamically generates a list of relevant "compatibility checker objects" (e.g., `MotorEscChecker`, `FcPeripheralVoltageChecker`) for all applicable pairs or sets within the input.
2. **Evaluation:** The engine then calls `.evaluate()` on each instantiated checker object.
3. **Aggregation:** The engine collects the `messages` returned from each checker and bundles them with the checker's name and the specific component subset it evaluated. The final output is an array of these grouped result objects.

This object-oriented pattern keeps rules encapsulated while allowing the frontend to send either pairwise checks or complete builds through the exact same pipeline.

---

## 4. Rules & Checkers

### 4.1. Electrical Compatibility
- **Voltage Limits:** Primary power (ESC, FC) checked against battery voltage. Peripheral power (VTX, Camera, RX, GPS) checked against FC BEC outputs first, then VBAT. (Outputs `DEFINITE_INCOMPATIBILITY`).
- **Motor vs. ESC Current:** Heuristic calculation of peak motor draw compared against ESC limits. (Outputs `HEURISTIC_INCOMPATIBILITY`).
- **Battery Discharge Limits:** Calculated total system draw compared against battery C-Rating capacity. (Outputs `HEURISTIC_INCOMPATIBILITY`).

### 4.2. Physical Compatibility
- **Propeller Clearance:** Frame `maxPropSizeMm` must be $\ge$ Propeller `diameterMm`. (Outputs `DEFINITE_INCOMPATIBILITY`).
- **Mount Patterns:**
  - **Stack:** Frame `BoardMountPattern` must intersect with FC, ESC, and VTX mount patterns (e.g., 20x20mm, 30.5x30.5mm).
  - **Motors:** Frame `MotorMountPattern` must intersect with Motor mount patterns (e.g., 16x16mm).
- **Camera Fit:** Camera `widthMm` vs Frame `cameraMountWidthMm`. (Outputs `DEFINITE_INCOMPATIBILITY`, or `PARTIAL_COMPATIBILITY` if known adapters exist).
- **Spatial / Stack Clearance:** Heuristic sum of FC, ESC, and VTX `boardHeightMm` vs the frame's available height.

### 4.3. Software & Ecosystem Compatibility
- **Video Ecosystem:** Camera, VTX, and Goggles must all share the same `VtxEcosystem` (e.g., DJI O3, Walksnail, Analog). (Outputs `DEFINITE_INCOMPATIBILITY`).
- **Camera-VTX Connection:** Matches the `videoConnectionStandard` between the Camera and VTX. For proprietary digital systems (MIPI cables), a mismatch is a `DEFINITE_INCOMPATIBILITY`. If both are "Analog", it passes but outputs an `INFORMATIONAL` Note reminding the user to wire them through the FC for OSD.
- **Control Link:** Receiver and Transmitter must share an intersecting `RfProtocol` (e.g., ExpressLRS, Crossfire).
- **Firmware Matching:** FC and ESC target firmwares (e.g., Betaflight, Bluejay, AM32).

### 4.4. RF & Antenna Compatibility
- **Frequency Matching:** VTX/Goggles and RX/TX must operate on the same `RfFrequency` (e.g., 5.8GHz, 2.4GHz, 900MHz).
- **Antenna Connectors:** VTX/RX connector type (e.g., SMA, MMCX, U.FL) must exactly match the selected Antenna's connector.
- **Antenna Polarization:** VTX antenna (e.g., RHCP, LHCP, Linear) must match Goggles antenna polarization. (Mismatch outputs `HEURISTIC_INCOMPATIBILITY` as it causes massive signal loss, though video still technically works at close range).

### 4.5. Database Overrides
- **Known Incompatibilities:** Queries the `IncompatibilityIssue` table. If the evaluated components trigger a specific override record (e.g., "This specific FC has a noisy gyro when paired with this specific ESC"), it outputs the issue's defined message and severity.

### 4.6. Peripheral Connectivity
- **UART Availability:** Calculates the total number of UARTs required by the selected peripherals:
  - **Receiver:** Typically requires 1 full UART.
  - **GPS:** Requires 1 full UART.
  - **Digital VTX:** Requires 1 full UART for OSD/MSP (Analog usually does not).
- **Validation:** Compares the total required UARTs against the Flight Controller's available `uartConnections`. (Outputs `DEFINITE_INCOMPATIBILITY` if there are not enough UARTs).
