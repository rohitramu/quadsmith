# <a id="build-evaluator-specification" >Build Evaluator Specification</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Goal](#1-goal)
- [2. Proposal](#2-proposal)
  - [Point-Mass Mass, Inertia, and Center of Gravity Modeling](#point-mass-mass-inertia-and-center-of-gravity-modeling)
  - [Analytical Aerodynamic Thrust and Voltage Sag Modeling](#analytical-aerodynamic-thrust-and-voltage-sag-modeling)
  - [Aerodynamic and Electrical Power Consumption Modeling](#aerodynamic-and-electrical-power-consumption-modeling)
  - [Multi-Profile Flight Time and Range Boundary Estimation](#multi-profile-flight-time-and-range-boundary-estimation)
  - [Handling Quality via Disk Loading and Torque-to-Inertia Ratios](#handling-quality-via-disk-loading-and-torque-to-inertia-ratios)
  - [Continuous Payload Capacity Curves and Threshold Markers](#continuous-payload-capacity-curves-and-threshold-markers)
  - [Experimental Designation and Safety Disclaimers](#experimental-designation-and-safety-disclaimers)
- [3. Assumptions](#3-assumptions)

---

## <a id="1-goal" >1. Goal</a>

Provide an analytical flight dynamics evaluation engine that computes estimated
flight performance, mass distribution, agility, flight times, and operational
limits for any arbitrary quadcopter build configuration paired with runtime
battery and payload parameters.

---

## <a id="2-proposal" >2. Proposal</a>

> **TL;DR:**
>
> - Estimate total mass, center of gravity offset, and 3D
>   [moments of inertia](https://en.wikipedia.org/wiki/Moment_of_inertia)
>   ($I_{xx}, I_{yy}, I_{zz}$) using discrete point-mass approximations.
> - Model battery
>   [voltage sag](https://en.wikipedia.org/wiki/Internal_resistance) and loaded
>   RPM to analytically evaluate static thrust and electrical power draw without
>   requiring empirical lookup tables.
> - Compute core flight metrics:
>   [thrust-to-weight ratio (TWR)](https://en.wikipedia.org/wiki/Thrust-to-weight_ratio),
>   aerodynamic top speed via the
>   [drag equation](https://en.wikipedia.org/wiki/Drag_equation), multi-style
>   flight times (hover, cruise, freestyle, race), and dual range limits (RF vs.
>   aerodynamic).
> - Assess flight agility and prop wash handling through
>   [disk loading](https://en.wikipedia.org/wiki/Disk_loading) and
>   torque-to-inertia ratios, accompanied by continuous payload vs. TWR curves.
> - Label all evaluator surfaces as `"Experimental"` and link directly to
>   hardware, electrical, and flight safety disclaimers (`/terms`).

The Build Evaluator acts as a pure domain logic service within the Quadsmith
platform. It ingests an assembled quadcopter hardware configuration (frame
geometry, motor stators and KV, propeller dimensions, component weights, and
transmission specifications) alongside runtime inputs (selected battery profile
and auxiliary payload mass like an action camera). The engine executes
analytical physics formulas to evaluate electrical, aerodynamic, and inertial
characteristics, producing a structured telemetry report for the frontend user
interface.

### <a id="point-mass-mass-inertia-and-center-of-gravity-modeling" >Point-Mass Mass, Inertia, and Center of Gravity Modeling</a>

The evaluator calculates total flying mass ($M_{total} = M_{dry} + M_{battery} +
M_{payload}$) and resolves the center of gravity (CG) along three axes. Core
electronics (Flight Controller, ESC, Receiver, Camera) are localized at the
origin $(0, 0, 0)$, while arm masses (motor plus propeller) are distributed at a
radial distance $r = \frac{\text{Wheelbase}}{2}$. Battery and auxiliary payload
masses are offset along the vertical Z-axis (top-mount vs. bottom-mount) and
longitudinal X-axis (forward/aft camera placement).

When forward payload placement shifts the center of gravity away from the
geometric center, the evaluator computes the required steady-state motor thrust
differential. Rear motors must generate higher baseline thrust to maintain level
pitch, diminishing available peak collective thrust. Agility in yaw ($I_{zz}$),
roll ($I_{xx}$), and pitch ($I_{yy}$) is approximated using discrete point-mass
inertia summations:

$$I_{zz} = 4 \times M_{arm} \times r^2 + I_{core}$$

<details>
<summary><strong>Alternatives considered</strong></summary>

Using full 3D CAD mesh geometry and finite element analysis (FEA) to compute
exact volume and mass distributions.

**Pros:**

- Delivers mathematically precise mass distribution and rotational inertia
  tensors.

**Cons:**

- Impractical for community-driven catalogs where manufacturers rarely publish
  complete 3D STEP or STL models.
- Computationally prohibitive for interactive web and API evaluations.

</details>

### <a id="analytical-aerodynamic-thrust-and-voltage-sag-modeling" >Analytical Aerodynamic Thrust and Voltage Sag Modeling</a>

To evaluate static thrust across diverse motor and propeller combinations, the
engine applies momentum theory and aerodynamic coefficients rather than static
empirical databases. Air density is evaluated at standard sea level ($\rho =
1.225 \text{ kg/m}^3$). Effective operating voltage is modeled by incorporating
internal battery cell resistance derived from capacity and discharge C-ratings,
determining loaded voltage sag from nominal $S \times 3.7\text{V}$:

$$\text{No-Load RPM} = KV \times V_{sagged}$$

Loaded operating RPM ($n$) is evaluated at 75%–85% of no-load RPM based on
propeller aerodynamic load factor. Static thrust ($T$) is determined by
propeller diameter ($D$), blade count, pitch, and thrust coefficient ($C_T$):

$$T = C_T \times \rho \times n^2 \times D^4$$

Configuration modifiers adjust static output: ducted propeller ducts grant a
$+15\%$ static thrust increase, while pusher configurations apply a $-5\%$
penalty due to arm wake turbulence.

<details>
<summary><strong>Alternatives considered</strong></summary>

Empirical bench test lookup tables compiled from physical thrust stand
measurements.

**Pros:**

- Captures real-world non-linearities and specific propeller blade airfoil
  efficiencies.

**Cons:**

- Fails completely for newly released, unmeasured, or uncommon motor-propeller
  combinations.
- Requires maintaining massive, brittle external test datasets.

</details>

### <a id="aerodynamic-and-electrical-power-consumption-modeling" >Aerodynamic and Electrical Power Consumption Modeling</a>

Mechanical power required to turn the propeller blades is calculated based on
pitch, diameter, blade count, rotational speed, and power coefficient ($C_P$):

$$P_{mech} = C_P \times \rho \times n^3 \times D^5$$

Electrical power consumption incorporates motor electromechanical conversion
efficiency ($\eta_{motor} \approx 0.80$):

$$P_{elec} = \frac{P_{mech}}{\eta_{motor}}$$

Current draw ($I$) is subsequently resolved as $I =
\frac{P_{elec}}{V_{sagged}}$, establishing baseline current consumption across
throttle ranges.

### <a id="multi-profile-flight-time-and-range-boundary-estimation" >Multi-Profile Flight Time and Range Boundary Estimation</a>

Flight durations are calculated simultaneously across four standardized flight
profiles rather than a single idealized hover figure:

- **Hover**: Solves for the exact RPM and current required to satisfy $4 \times
  T = M_{total}$. $\text{Time} = \frac{\text{Capacity}}{\text{Hover Current}}$.
- **Cruising / Cinematic**: Evaluated at $1.5 \times \text{Hover Current}$.
- **Freestyle**: Evaluated at $3.0 \times \text{Hover Current}$, simulating
  aggressive throttle bursts alternating with zero-throttle inverted maneuvers.
- **Racing**: Evaluated at $0.4 \times \text{Max Current}$, simulating sustained
  high-speed forward flight.

Operational range is reported as the conservative minimum of two distinct
limits:

1. **Aerodynamic Endurance Range**: Forward cruise speed multiplied by cruising
   flight duration ($v_{cruise} \times \text{Time}_{cinematic}$).
2. **RF Telemetry Boundary**: Calculated from video transmitter output power
   (mW) and receiver protocol link budget (e.g., 25mW analog $\approx
   500\text{m}$, 800mW digital $\approx 4\text{km}$, 2.4GHz ExpressLRS $\approx
   15\text{km}$).

### <a id="handling-quality-via-disk-loading-and-torque-to-inertia-ratios" >Handling Quality via Disk Loading and Torque-to-Inertia Ratios</a>

To evaluate handling crispness and prop wash vulnerability, the engine
calculates two fundamental physical ratios:

- **Disk Loading ($DL$)**: Ratio of total mass to total propeller swept disk
  area ($\frac{M_{total}}{\pi \times r_{prop}^2 \times 4}$). Lower disk loading
  corresponds to superior grip, float, and prop wash recovery.
- **Torque-to-Inertia Ratio**: Motor stator volume (serving as an analytical
  proxy for available magnetic torque) divided by yaw moment of inertia
  ($I_{zz}$).

These metrics are reported as raw physical dimensions ($\text{g/cm}^2$)
alongside a normalized dimensionless **Handling Score (1–10)**.

### <a id="continuous-payload-capacity-curves-and-threshold-markers" >Continuous Payload Capacity Curves and Threshold Markers</a>

Rather than computing a single payload cutoff weight, the engine models payload
capacity as a continuous curve of auxiliary payload mass ($0\text{g}$ to
$1000\text{g}$) against resulting thrust-to-weight ratio. The report overlays
standardized benchmark thresholds:

- **Cinematic Benchmark**: $2.0:1$ TWR (smooth, stable camera carrying).
- **Freestyle Benchmark**: $4.5:1$ TWR (punchy acceleration and acrobatic
  authority).
- **Absolute Usability Floor**: $1.5:1$ TWR (minimum threshold for reliable
  outdoor flight control).

### <a id="experimental-designation-and-safety-disclaimers" >Experimental Designation and Safety Disclaimers</a>

All Build Evaluator surfaces across the web application are explicitly labeled
with an **`Experimental`** badge. They state that the Build Evaluator is
provided `"AS IS"` without warranties of airworthiness or safety:

- **Build Profile Dashboard**
  ([`BuildProfilePage.tsx`](../src/react/app-web/src/pages/BuildProfilePage.tsx)):
  Displays the `Experimental` badge alongside `Build Evaluation & Performance`,
  states that the Build Evaluator is provided `"AS IS"` without warranties of
  airworthiness or safety, and opens the safety disclaimers modal when
  `Safety disclaimers` is clicked.
- **The Forge Live Evaluator**
  ([`BuildWizardPage.tsx`](../src/react/app-web/src/pages/BuildWizardPage.tsx)):
  Displays the `Experimental` badge and `Safety disclaimers` modal trigger in
  both the desktop evaluator card and the mobile collapsible telemetry drawer.
  Step 12
  ([`ReviewStep.tsx`](../src/react/app-web/src/components/forge/steps/ReviewStep.tsx))
  also displays a pre-save hardware, flight safety, and local compliance notice.
- **Build Cards & WASM Engine Loader**
  ([`BuildCard.tsx`](../src/react/app-web/src/components/BuildCard.tsx),
  [`WasmLoadingIndicator.tsx`](../src/react/app-web/src/components/WasmLoadingIndicator.tsx)):
  Marks quick-evaluated card outputs and the WebAssembly loader as
  `Experimental` and provided `"AS IS"` without warranties of airworthiness or
  safety.
- **Safety Disclaimers Modal & Terms of Service Page**
  ([`SafetyDisclaimersModal.tsx`](../src/react/app-web/src/components/SafetyDisclaimersModal.tsx),
  [`TermsOfServicePage.tsx`](../src/react/app-web/src/pages/TermsOfServicePage.tsx)):
  Clicking `Safety Disclaimers` anywhere in the application opens a modal dialog
  covering LiPo electrical safety, mechanical and propeller safety, compliance
  with local rules and regulations, and manufacturer specification verification,
  alongside the full `/terms` page.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="blocking-modal-disclaimer-on-every-build-evaluation" >Blocking Modal Disclaimer on Every Build Evaluation</a>

Display a mandatory click-through modal dialog before allowing users to view any
Build Profile or open The Forge configurator.

**Pros:**

- Guarantees every user clicks an acknowledgment button before seeing flight
  estimates.

**Cons:**

- Degrades browsing and configuration UX with repetitive modal friction on every
  session.
- Persistent inline `Experimental` badges, on-demand `Safety Disclaimers` modal
  dialogs, and the `/terms` page communicate the exact boundaries without
  blocking read-only exploration.

</details>

---

## <a id="3-assumptions" >3. Assumptions</a>

1. Air density ($\rho$) is evaluated at standard International Standard
   Atmosphere (ISA) sea-level conditions ($1.225 \text{ kg/m}^3$).
2. Nominal battery cell voltage is assumed to be $3.7\text{V}$ per cell ($S
   \times 3.7\text{V}$).
3. Brushless motor electromechanical energy conversion efficiency is modeled at
   approximately 80% ($\eta_{motor} = 0.80$).
4. Loaded motor RPM under aerodynamic drag ranges between 75% and 85% of
   theoretical no-load RPM depending on propeller pitch and diameter load
   factors.
5. Component masses and dimensional geometry entered in catalog records reflect
   true physical hardware values within reasonable manufacturing tolerances.
