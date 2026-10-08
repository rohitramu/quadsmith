# Build Evaluator Specification

## 1. Overview
The Build Evaluator is a logic module that takes a Quadcopter `Build` configuration and a runtime `Battery` profile, and calculates estimated flight characteristics. This data is used to generate a detailed performance report for the user.

## 2. Inputs

### 2.1 Hardware Configuration (from Database)
- **Frame**: Wheelbase (mm), Configuration (Puller/Pusher, Ducted/Un-ducted).
- **Motors**: KV, Stator Size (Diameter mm, Height mm).
- **Propellers**: Diameter (in), Pitch (in), Blade Count.
- **Components**: FC, ESC, Video Transmitter, RX, Camera, GPS (used for accumulating Dry Weight).
- **Video Transmitter**: Max Power (mW).
- **Receiver**: Protocol (e.g., ELRS, Crossfire, SBUS).

### 2.2 Runtime Parameters (User Input)
- **Battery Component ID**: Selected from the database (provides Capacity, Cell Count, Weight, and Discharge Rate).
- **Payload Weight (g)**: Extra mass not part of the core build (e.g., a GoPro or cinematic camera).
- **Flight Style** (Used for some specific metric highlights, though we calculate all profiles).

---

## 3. Core Physics Models

### 3.1 Mass, Inertia & CG Model
- **Total Mass ($M$)**: $M_{dry} + M_{battery} + M_{payload}$
- **Center of Gravity (CG) Offset**: If a heavy payload is mounted far forward, the CG shifts. The evaluator calculates the necessary thrust differential where rear motors must work harder to maintain hover, effectively reducing total max usable thrust.
- **Inertia Tensor ($I_{xx}, I_{yy}, I_{zz}$)**:
  Uses a point-mass approximation for full 3D agility estimates.
  - **Core Mass ($M_{core}$)**: Sum of Frame, FC, ESC, Video Transmitter, RX, Camera. Located at center $(0,0,0)$.
  - **Battery & Payload Mass**: Offset along the Z-axis (top vs bottom mount) and X-axis (forward/back), significantly impacting Pitch ($I_{yy}$) and Roll ($I_{xx}$) inertia.
  - **Arm Mass ($M_{arm}$)**: Motor + Propeller weight.
  - **Distance ($r$)**: $\frac{\text{Wheelbase}}{2}$.
  - **Approximate Inertias**:
    - Yaw ($I_{zz}$): $4 \times M_{arm} \times r^2 + I_{core\_approx}$
    - Roll/Pitch ($I_{xx}, I_{yy}$): Calculated based on the vertical/horizontal spread of the battery and payload relative to the motor plane.

### 3.2 Thrust Model (Analytical)
Since empirical data is not available for all combos, we use an analytical aerodynamic model.
- **Air Density ($\rho$)**: We assume standard sea-level air density ($1.225 \text{ kg/m}^3$) for all calculations.
- **Voltage ($V$) & Sag**: Starts at $S \times 3.7\text{V}$ (Nominal). Under heavy load, the evaluator models **Voltage Sag** caused by the battery's Internal Resistance (estimated via C-rating and capacity). The *true* voltage under load is used for max thrust calculation.
- **No-Load RPM**: $KV \times V_{sagged}$.
- **Loaded RPM ($n$)**: Estimated at ~75-85% of No-Load RPM depending on prop load factor.
- **Static Thrust ($T$)**:
  $T = C_T \times \rho \times n^2 \times D^4$
  Where $C_T$ (Thrust Coefficient) is derived from Pitch and Blade Count.
- **Configuration Modifiers**:
  - *Ducted*: +15% static thrust, but increased aerodynamic drag at speed.
  - *Pusher*: -5% thrust if obstructed by arms, but cleaner air to props.

### 3.3 Power Consumption Model
- **Mechanical Power ($P_{mech}$)**: $C_P \times \rho \times n^3 \times D^5$
- **Electrical Power ($P_{elec}$)**: $\frac{P_{mech}}{\eta_{motor}}$ (Assume ~80% motor efficiency).
- **Current Draw ($I$)**: $\frac{P_{elec}}{V}$.

---

## 4. Evaluated Metrics

### 4.1 Thrust-to-Weight Ratio (TWR)
- **Calculation**: $\text{TWR} = \frac{4 \times T_{max}}{M_{total}}$
- **Output**: A simple ratio (e.g., 6.5:1).

### 4.2 Max Speed
- **Pitch Speed**: $n \times \text{Pitch}$. This is the theoretical maximum speed of the air leaving the prop.
- **Aerodynamic Drag ($F_D$)**: Estimated frontal area drag. $F_D = \frac{1}{2} \rho v^2 C_D A$.
- **Calculation**: Max speed $v_{max}$ is reached when forward thrust component equals drag. Ducted frames will have a higher $C_D$, significantly lowering $v_{max}$.

### 4.3 Flight Time
Calculated for multiple flight styles simultaneously:
1. **Hover**: Calculate the RPM required to generate $T = \frac{M_{total}}{4}$. Find the power and current at this RPM. $\text{Time} = \frac{\text{Capacity}}{\text{Hover Current}}$.
2. **Cinematic / Cruising**: $\text{Hover Current} \times 1.5$.
3. **Freestyle**: $\text{Hover Current} \times 3.0$ (Aggressive throttle punches mixed with zero-throttle hangtime).
4. **Racing**: $\text{Max Current} \times 0.4$ (Sustained high throttle).

### 4.4 Range
Presented as two separate bottlenecks:
- **Aerodynamic Range**: $v_{cruise} \times \text{Flight Time}_{cinematic}$.
- **Estimated RF Range**: Based on Video Transmitter Max Power and Protocol. (e.g., 25mW Analog ~500m, 800mW Digital ~4km, ELRS 2.4GHz ~15km). The practical range is the minimum of Video and Control limits.

### 4.5 Prop Wash Handling Quality
- **Disk Loading ($DL$)**: $\frac{M_{total}}{\text{Total Swept Area}}$ (Lower is better for grip and float).
- **Torque-to-Inertia Ratio**: Motor stator volume (proxy for torque) divided by $I_{zz}$. Represents how quickly the motors can change the drone's attitude.
- **Output**: The report will display a dimensionless **Handling Score (1-10)** alongside the raw physical metrics ($DL$ in g/cm² and the Torque/Inertia ratio).

### 4.6 Max Payload Capacity
- Evaluated as a continuous chart rather than a single number.
- **X-Axis**: Payload Weight (g).
- **Y-Axis**: Resulting TWR.
- **Markers**: 
  - Recommended TWR for Cinematic (e.g., 2.0:1)
  - Recommended TWR for Freestyle (e.g., 4.5:1)
  - Absolute Minimum (1.5:1)

---

## 5. UI / UX Flow
1. **View Build Page**: User opens a Build.
2. **Evaluation Tab**: User clicks "Evaluate".
3. **Runtime Input**: User selects a Battery from the database (via Component ID) and optionally inputs an additional "Payload Weight" (e.g., for a GoPro).
4. **Report Generation**: The evaluator runs the math and displays a dashboard with the metrics defined above.
