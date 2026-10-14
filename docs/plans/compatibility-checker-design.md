# <a id="compatibility-checker-architecture" >Compatibility Checker Architecture</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Goal](#1-goal)
- [2. Proposal](#2-proposal)
  - [Permissive Non-Blocking Build Persistence](#permissive-non-blocking-build-persistence)
  - [Stateless Arbitrary Component Evaluation API](#stateless-arbitrary-component-evaluation-api)
  - [Modular Rule Checker Encapsulation and Telemetry Grouping](#modular-rule-checker-encapsulation-and-telemetry-grouping)
  - [Physical Dimensional Fit Rules](#physical-dimensional-fit-rules)
  - [Electrical and Thermal Power Constraints](#electrical-and-thermal-power-constraints)
  - [Software, Firmware, and Ecosystem Compatibility](#software-firmware-and-ecosystem-compatibility)
  - [RF Protocol and Antenna Compatibility](#rf-protocol-and-antenna-compatibility)
  - [Hardware Overrides and Peripheral UART Budgeting](#hardware-overrides-and-peripheral-uart-budgeting)
- [3. Assumptions](#3-assumptions)

---

## <a id="1-goal" >1. Goal</a>

Provide an extensible, backend-driven compatibility evaluation engine that
analyzes arbitrary combinations of quadcopter components, verifying physical
fitment, electrical boundaries, software ecosystems, and peripheral
interconnects while returning rich numerical telemetry and actionable resolution
guidance.

---

## <a id="2-proposal" >2. Proposal</a>

> **TL;DR:**
>
> - Evaluate arbitrary sets of components statelessly on the backend without
>   requiring a pre-existing saved build.
> - Emit structured result objects grouping severity-coded messages, actionable
>   resolution tips, and typed metrics for UI gauges.
> - Verify physical dimensions (stack mounts, propeller clearance, camera
>   width), electrical limits (voltage ranges, current headroom), software
>   protocols (video links, control links), and UART peripheral budgets.
> - Permit permissive saving of builds flagged with definite incompatibilities
>   to support custom hardware modifications.

The Compatibility Checker operates as a centralized backend evaluation service.
Rather than embedding validation logic across frontend interfaces, the backend
serves as the single source of truth. The engine dynamically instantiates
specialized checker objects based on the input component selection, executes
deterministic validation checks, and returns categorized diagnostics alongside
typed telemetry metrics that drive rich visual components in the user interface.

### <a id="permissive-non-blocking-build-persistence" >Permissive Non-Blocking Build Persistence</a>

When the compatibility engine identifies issues flagged with
`DEFINITE_INCOMPATIBILITY`, the platform highlights the errors prominently in
the UI but allows the user to save the build configuration. FPV builders
routinely deploy 3D-printed adapter brackets, custom carbon drillings,
soft-mount spacers, and bespoke wiring harnesses that overcome catalog physical
or electrical constraints. The system acts as an advisory assistant rather than
a restrictive gatekeeper.

#### <a id="alternatives-considered-permissive-build-persistence" >Alternatives considered</a>

Strict persistence gating that rejects saving or publishing builds containing
unresolved `DEFINITE_INCOMPATIBILITY` diagnostics.

**Pros:**

- Guarantees that every build stored in the public registry is guaranteed to be
  physically and electrically operable out of the box.

**Cons:**

- Blocks valid custom hardware builds, prototype work, and work-in-progress
  configurations.
- Frustrates users when catalog specifications contain minor discrepancies or
  unlisted hardware tolerances.

### <a id="stateless-arbitrary-component-evaluation-api" >Stateless Arbitrary Component Evaluation API</a>

The backend exposes a standalone endpoint (e.g.,
`POST /api/compatibility/check`) accepting an arbitrary array of component IDs.
The endpoint does not require a persistent `Build` entity, allowing client
applications to run lightweight pre-selection checks (such as verifying if a
specific motor fits a chosen frame before adding it to an aircraft). The backend
executes a single batch query to resolve specifications and known override
issues, runs the evaluation in memory, and returns the diagnostics payload.

#### <a id="alternatives-considered-stateless-evaluation-api" >Alternatives considered</a>

Requiring a persistent `Build` record in the database before running
compatibility checks.

**Pros:**

- Direct foreign key joins on existing build tables without handling loose
  component arrays.

**Cons:**

- Prevents real-time compatibility feedback during the component selection stage
  in the frontend builder.
- Disallows ad-hoc pairwise component checking (e.g., comparing a motor and an
  ESC in isolation).

### <a id="modular-rule-checker-encapsulation-and-telemetry-grouping" >Modular Rule Checker Encapsulation and Telemetry Grouping</a>

The evaluation pipeline accepts an arbitrary list of components and dynamically
instantiates modular checker instances (e.g., `MotorEscChecker`,
`FcPeripheralVoltageChecker`). Each checker returns an evaluation result
containing:

- **`components`**: The subset of evaluated components.
- **`checkerName`**: Explicit identifier for the executing rule.
- **`metrics`**: Typed numerical and categorical telemetry (e.g.,
  `fcMountMatch: { expected: ["20x20"], actual: ["30.5x30.5"] }`).
- **`messages`**: Array of issues containing severity levels
  (`DEFINITE_INCOMPATIBILITY`, `HEURISTIC_INCOMPATIBILITY`,
  `PARTIAL_COMPATIBILITY`, `INFORMATIONAL`), human-readable descriptions, and
  concrete remediation steps.

### <a id="physical-dimensional-fit-rules" >Physical Dimensional Fit Rules</a>

The engine enforces mechanical boundary constraints across hardware components:

- **Propeller Clearance**: Verifies that the frame's `maxPropSizeMm` satisfies
  the propeller's `diameterMm`.
- **Mount Patterns**: Ensures frame `BoardMountPattern` sets intersect with
  Flight Controller, ESC, and Video Transmitter mounting patterns (e.g.,
  $20\text{mm} \times 20\text{mm}$, $30.5\text{mm} \times 30.5\text{mm}$), and
  frame `MotorMountPattern` sets match motor base hole patterns (e.g.,
  $16\text{mm} \times 16\text{mm}$, $12\text{mm} \times 12\text{mm}$).
- **Camera Bracket Fit**: Compares camera chassis width against the frame's
  camera mounting bracket spacing, emitting `PARTIAL_COMPATIBILITY` if standard
  adapter brackets exist.
- **Stack Height Clearance**: Sums component board thicknesses and heatsink
  profiles against available vertical frame standoff clearance.

### <a id="electrical-and-thermal-power-constraints" >Electrical and Thermal Power Constraints</a>

Power supply and current tolerances are verified across the power distribution
tree:

- **Voltage Limits**: Verifies nominal and maximum battery voltage against input
  voltage ratings for the ESC, Flight Controller, and VTX. Checks peripheral
  operating voltages against Flight Controller onboard Battery Elimination
  Circuit (BEC) outputs before falling back to raw battery voltage ($V_{BAT}$).
- **Motor vs. ESC Current**: Evaluates maximum expected motor burst current
  against ESC continuous and burst ratings, emitting `HEURISTIC_INCOMPATIBILITY`
  if ESC limits are exceeded.
- **Battery Discharge Capacity**: Compares estimated total multi-motor draw
  against battery rated continuous discharge capacity ($C \times
  \text{Capacity}$).

### <a id="software-firmware-and-ecosystem-compatibility" >Software, Firmware, and Ecosystem Compatibility</a>

Digital interconnects and firmware configurations are validated for protocol
parity:

- **Video Transmission Ecosystem**: Ensures the camera, video transmitter, and
  goggles operate within matching digital or analog ecosystems (e.g., DJI O3,
  Walksnail Avatar, HDZero, or Analog).
- **Video Signal Interfacing**: Verifies the physical cable connection (e.g.,
  MIPI digital ribbon vs. composite analog video). If analog, emits an
  `INFORMATIONAL` notice recommending routing video signals through the flight
  controller for On-Screen Display (OSD) overlay.
- **Control Link Protocols**: Confirms the radio receiver and handheld radio
  transmitter share a compatible radio frequency protocol (e.g., ExpressLRS, TBS
  Crossfire).
- **Firmware Ecosystems**: Validates interoperability between flight controller
  firmware targets and ESC communication protocols (e.g., DShot300/600 on
  Betaflight, Bluejay, or AM32).

### <a id="rf-protocol-and-antenna-compatibility" >RF Protocol and Antenna Compatibility</a>

Radio transmission and reception interfaces must align across spectrum and
connector mechanics:

- **Operating Frequency**: Ensures video transmitters and goggles, as well as
  transmitters and receivers, share matched operational bands (e.g., 5.8GHz,
  2.4GHz, 900MHz).
- **Antenna Connector Interfaces**: Matches female/male connector standards
  (e.g., SMA, RP-SMA, U.FL/IPEX, MMCX).
- **Antenna Polarization**: Compares video transmitter antenna polarization
  (RHCP, LHCP, or Linear) against receiver goggles antennas. Polarization
  mismatches emit `HEURISTIC_INCOMPATIBILITY` due to severe attenuation ($>
  20\text{ dB}$ cross-polarization rejection).

### <a id="hardware-overrides-and-peripheral-uart-budgeting" >Hardware Overrides and Peripheral UART Budgeting</a>

To account for hardware revisions, silicon quirks, and bus contention:

- **Catalog Incompatibility Overrides**: Queries the database
  `IncompatibilityIssue` table for documented component pairings with known
  hardware conflicts (e.g., electrical noise on specific gyro revisions when
  paired with unbuffered ESCs).
- **UART Allocation Budget**: Tallies required serial UART peripherals across
  connected devices (Receiver $\approx 1$, GPS $\approx 1$, Digital VTX MSP/OSD
  $\approx 1$) and validates against the flight controller's available hardware
  `uartConnections`.

---

## <a id="3-assumptions" >3. Assumptions</a>

1. Compatibility rules execute purely in backend memory using fetched catalog
   specifications and override rules.
2. Component specifications accurately reflect manufacturer tolerances, pinouts,
   and physical mount measurements.
3. Permissive build persistence allows builders to document customized hardware
   modifications without system rejection.
