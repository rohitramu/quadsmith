# Hardware Taxonomy and Build Domain Model

## 1. Overview
This document outlines the conceptual domain model for Quadsmith's hardware catalog and build configurations. The architecture is designed using Domain-Driven Design (DDD) principles to handle the complex, rapidly evolving nature of FPV drone components while maintaining clean boundaries for community features like reviews, Q&A, and compatibility checking. This model is independent of the underlying persistence layer.

## 2. Key Domain Design Decisions

### 2.1 Classical Inheritance for Component Taxonomy
**Decision:** The domain object model uses classical inheritance. Specific hardware and software profiles (e.g., `Motor`, `Frame`, `FcFirmware`) inherit from intermediate classes (`HardwareComponent` or `SoftwareComponent`), which in turn inherit from the base `Component` identity. 

**Rationale:** By having all parts inherit a unified `Component` identity, we implement global relations cleanly. Community features like `Reviews`, `Questions`, `Collections`, and `ReferenceLinks` can safely reference a single Component ID. A `Review` object doesn't need to know if it is reviewing a Motor or a Frame; it simply points to the base `Component` ID. 

### 2.2 Reusable Sub-Assemblies (True Aggregate Roots)
**Decision:** Electronics are grouped into intermediate standalone Aggregate Roots (`FlightStack` for FC+ESC, `VtxConfiguration` for VTX+Antennas) rather than being embedded by-value into the root `Build` entity.

**Rationale:** In our domain, a `FlightStack` is a distinct entity with its own unique identity (ID). A `Build` merely stores a reference to a `FlightStack ID`. This allows a user to define a trusted VTX+Antenna loadout once, and reference that exact same loadout across multiple builds. If the loadout is updated, it updates everywhere it is referenced, mirroring real-world modularity.

### 2.3 Strict Relational Entities for Managed Dictionaries
**Decision:** Hardware characteristics like `RfProtocol` and `BoardMountPattern` are modeled as strict Relational Entities (with unique IDs), rather than freeform strings, validation lists, or static enums. Hardware objects hold explicit references (e.g., `protocolId`) to these entities.

**Rationale:** Perfect data integrity is required for the Compatibility Checker to function. Freeform tags would result in data fragmentation (e.g., "elrs" vs "ExpressLRS"), breaking compatibility logic. Static enums would require a code deployment every time a new protocol is released. Relational Entities allow administrators to dynamically add new protocols at runtime, attach extra metadata to the protocol itself, and ensure all hardware references the exact same concept. To mitigate the administrative bottleneck of requiring formal entity creation, the platform relies on a "Suggest Edit" workflow, allowing the community to queue new dictionary terms for rapid admin approval without breaking data integrity.

### 2.4 Hierarchical Software Versioning
**Decision:** Software is modeled using a Parent-Child hierarchy. The `Software` entity represents the parent product family (e.g., "Betaflight"), which owns a collection of `SoftwareComponent` child entities representing specific compiled flavors of a version (e.g., "v4.3 - Target STM32F4"). Crucially, only the `SoftwareComponent` inherits from the base `Component` identity.

**Rationale:** This hierarchical approach accounts for the fact that a single software version often comes in many different compiled flavors or hardware targets. It allows users to view aggregate data (like total reviews or Q&A) at the parent product level ("Betaflight"), while still providing pinpoint accuracy for `IncompatibilityIssues` and specific build reviews, which target the `SoftwareComponent` entity directly (e.g., "Build Flavor X breaks compatibility with this specific ESC"). It also creates perfect symmetry with hardware: both `HardwareComponent` and `SoftwareComponent` represent the actual, installable artifacts that participate in the global component system.

---

## 3. Domain Architecture Breakdown

### 3.1 Base Identity (`Component` & `Company`)
All parts in the system inherit the base abstract `Component` identity. This encapsulates universal metadata:
- **Name & Release Date**
- **Associated `Company`** (Bidirectional association via a junction entity)
- **Reference Links** (Manuals, Purchase URLs)

### 3.2 Hardware Components
The `HardwareComponent` entity inherits from `Component` and introduces physical attributes like `weightG`. It serves as the parent class for concrete hardware classes:
- `Frame`, `Motor`, `Propeller`, `FlightController`, `Esc`, `Battery`, `Vtx`, `Camera`, `Receiver`, `Gps`, `Antenna`, `Transmitter`, `Goggles`.

Each subclass holds attributes unique to its category (e.g., `kV` for Motors, `wheelbaseMm` for Frames).

### 3.3 Software & Software Components
The `Software` entity represents a parent product family (e.g., "Betaflight"). It is a standalone entity that does *not* inherit from the global `Component` identity.

Each `Software` product contains a collection of `SoftwareComponent` entities. The `SoftwareComponent` entity inherits from the base `Component` table, giving it a global identity for reviews and compatibility checks. It serves as the parent class for specific firmware categories:
- `FcFirmware`, `EscFirmware`, `VtxFirmware`, `OperatingSystem`.

### 3.4 Managed Dictionaries (Relational Taxonomy)
Dynamic standardizations maintained at runtime as strict entities:
- **`VtxEcosystem`**: e.g., DJI, Walksnail, HDZero, Analog
- **`RfFrequency`**: e.g., 2.4GHz, 5.8GHz, 900MHz
- **`RfProtocol`**: e.g., ELRS, Crossfire
- **`AntennaPolarization`**: e.g., RHCP, LHCP, Linear
- **`AntennaConnector`**: e.g., SMA, UFL, MMCX
- **`MotorMountPattern` & `BoardMountPattern`**: e.g., 30.5x30.5, 20x20

### 3.5 The Aircraft (`Build` Root Aggregate)
The `Build` entity is the Root Aggregate representing a user's complete quadcopter. 
- **Direct References:** Holds IDs to `Frame`, `Motor`, `Propeller`, `Camera`, `Gps`.
- **Sub-Assembly References:** Holds IDs to `FlightStack`, `VtxConfiguration`, `ReceiverConfiguration`.
- **Metadata:** `name`, `crashResistanceRating`, `miscWeightG`, `isVerified`.
- **Taxonomy:** `Tags` (e.g., "freestyle", "cinewhoop").

Like `Components`, `Builds` participate in global community relations, serving as referenceable targets for `Reviews` and `Questions`.
