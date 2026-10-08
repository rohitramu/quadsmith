# Quadsmith Web UI Testing Plan: Rendering & Navigation

## 1. Overview & Objectives

This document establishes the UI testing strategy and implementation plan for the Quadsmith web frontend (`src/frontend/web`).

The primary objectives are:
1. **Rendering Integrity**: Verify that all core UI components, pages, data tables, specification cards, and interactive controls render accurately across various states (loading, loaded, empty, error, not found).
2. **Navigation Reliability**: Ensure seamless routing and navigation across all views (top navbar, collapsible sidebar, category grids, data table row deep links, pagination controls, and multi-level breadcrumbs).
3. **Developer Velocity & CI Safety**: Maintain sub-second test execution speeds and hermetic isolation so tests run reliably locally and are integrated directly into `make test`.

---

## 2. Testing Architecture & Decisions

### 2.1 Component & Route Integration Suite (Vitest + React Testing Library + JSDOM)

For unit, component, and multi-page route navigation tests, the testing stack utilizes:
- **Test Runner**: [Vitest](https://vitest.dev/) configured for Vite and TypeScript.
- **DOM Environment**: `jsdom` for headless in-memory browser simulation.
- **Component Testing**: `@testing-library/react` and `@testing-library/user-event` for user-centric interaction testing.
- **DOM Assertions**: `@testing-library/jest-dom` for expressive DOM state matchers.

#### Key Architectural Benefits:
- **Sub-Second Execution**: Tests run in milliseconds without requiring browser binary launches or Docker containers.
- **Type-Safe RPC Mocking**: Leveraging ConnectRPC's `createRouterTransport` from `@connectrpc/connect`, we provide strongly typed, in-memory service mocks for queries like `ListMotors`, `GetMotor`, `ListFrames`, etc., without network traffic or fragile HTTP monkeypatching.
- **Deep Routing Emulation**: Using `MemoryRouter` to test deep links, parameter resolution, breadcrumbs, and browser history navigation.

```mermaid
flowchart TD
    subgraph UI Testing Architecture
        subgraph In-Memory Test Suite (Vitest + RTL + JSDOM)
            V[Vitest Test Runner]
            MockTransport[Mock ConnectRPC RouterTransport]
            QueryClient[Test TanStack QueryClient]
            MemRouter[React Router MemoryRouter]
            
            V --> MockTransport
            V --> QueryClient
            V --> MemRouter
            
            MemRouter --> LayoutTests[Layout & Theme Tests]
            MemRouter --> NavFlowTests[Full Navigation Flow Tests]
            MemRouter --> CollectionTests[Collection Table, Filter & Columns]
            MemRouter --> ProductTests[Product Details & Specs]
        end

        subgraph Future E2E Suite (Playwright in test/web)
            PW[Playwright Headless Browser]
            PreviewServer[Vite Preview / Go Backend Sandbox]
            PW --> PreviewServer
        end

        In-Memory Test Suite --> MakeTest["make test (Enforced in local dev & CI)"]
        Future E2E Suite --> MakeE2E["make test-e2e (Full Browser Integration)"]
    end
```

---

## 3. Implementation Workstreams

### Workstream 1: Testing Infrastructure & Test Utilities

1. **Dependencies**:
   - Add devDependencies to `src/frontend/web/package.json`: `vitest`, `@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`, `jsdom`.
   - Add test scripts to `package.json`:
     - `"test": "vitest run"`
     - `"test:watch": "vitest"`
     - `"test:coverage": "vitest run --coverage"`
2. **Configuration (`vitest.config.ts`)**:
   - Merge with `vite.config.ts`.
   - Set `environment: "jsdom"`.
   - Configure setup files: `src/test/setup.ts`.
3. **Environment Setup (`src/frontend/web/src/test/setup.ts`)**:
   - Register `@testing-library/jest-dom/vitest`.
   - Clean up DOM and reset `localStorage` after each test run.
   - Mock browser APIs not natively provided in JSDOM (e.g., `window.matchMedia`).
4. **Mock Transport & Fixtures (`src/frontend/web/src/test/mocks/`)**:
   - Create realistic protobuf message fixtures for Motors, Frames, Flight Controllers, etc.
   - Implement `createRouterTransport` handlers for component services to serve mock responses for listing, filtering, pagination, and detail queries.
5. **Render Utilities (`src/frontend/web/src/test/test-utils.tsx`)**:
   - `renderWithProviders(ui, options)`: Helper wrapping components with `TransportProvider`, `QueryClientProvider`, and `MemoryRouter`.
   - `renderApp(initialRoute, options)`: Mounts the full application routes starting at `initialRoute`.

---

### Workstream 2: Application Decoupling for Testability

To enable both production browser execution and isolated in-memory route testing:
- **`src/frontend/web/src/App.tsx`**:
  - Extract and export `AppRoutes` containing the `<Routes>` hierarchy.
  - Allow `App` to accept optional `transport` and `queryClient` props.
  - This allows tests to mount `<AppRoutes />` within a `MemoryRouter` and custom mock transport, while production uses `<BrowserRouter>` and the live backend transport.

---

### Workstream 3: Test Suites

#### 3.1 Layout & Theme Navigation (`Layout.test.tsx`)
- **Header**:
  - Logo rendering and link to `/`.
  - Search input presence and placeholder.
  - Theme Toggle: Verifies clicking the Sun/Moon button toggles the `dark` class on `<html>`, updates `localStorage`, and swaps the theme icon.
- **Sidebar**:
  - Expanding and collapsing the "Hardware" menu section via the chevron toggle.
  - Verifying all 11 hardware collection links render with valid targets (`/components/hardware/motors`, etc.).
  - Verifying "Software" and "Gear" navigation links.

#### 3.2 Home Page (`HomePage.test.tsx`)
- Heading, welcome copy, and intro cards.
- "Browse Hardware" card links to `/components/hardware`.
- "Build Planner (Coming Soon)" placeholder rendering.

#### 3.3 Category Page (`CategoryPage.test.tsx`)
- Heading and grid of hardware collection cards (Motors, Frames, Batteries, Propellers, Cameras, VTX, Antennas, Receivers, Flight Controllers, ESCs, GPS).
- Clicking collection cards triggers navigation to `/components/hardware/:collectionId`.
- Empty state message when accessing an undefined category.

#### 3.4 Collection Table & Controls (`CollectionPage.test.tsx`)
- **Breadcrumbs**: Hierarchical navigation (`Hardware > Motors`), with back-links.
- **States**: Loading indicators, empty results banner, and error state display.
- **Table Data**: Header labels, custom cell formatters (e.g. stator size, KV), and clickable rows linking to product details.
- **Column Customization (`ColumnSelector`)**:
  - Popover open/close behavior.
  - Checkbox toggling to hide or show columns.
  - Up/Down arrows to reorder visible columns.
  - "Reset to Default" restoration.
- **Sorting**: Header sort button clicks cycling through ascending (`field`), descending (`^field`), and neutral.
- **Pagination**: Page size selector (10, 20, 50, 100), page number display, Next/Previous page buttons, and token-based pagination state.
- **SmartFilterInput**: Filter input typing, preset suggestions, and applying filter queries.

#### 3.5 Product Detail Page (`ProductPage.test.tsx`)
- Breadcrumb navigation (`Hardware > Motors > [Product Name]`).
- Spec highlight cards (Stator Size, KV, Weight, Max Thrust).
- Technical specifications table key-value rendering.
- Reference links with external link icons and correct labels (Purchase, Official Product Page, Review, etc.).
- Fallback for nonexistent or not-found products.

#### 3.6 Full Navigation Flow (`NavigationFlow.test.tsx`)
Integration test validating an entire continuous user flow:
1. Start at `/` (Home).
2. Click "Browse Hardware" -> Navigates to `/components/hardware`.
3. Click "Motors" collection card -> Navigates to `/components/hardware/motors`.
4. Click row for "EMAX ECO II 2207" -> Navigates to `/components/hardware/motors/emax-eco-ii-2207`.
5. Click "Motors" in breadcrumbs -> Returns to `/components/hardware/motors`.
6. Click "Hardware" in sidebar -> Returns to `/components/hardware`.

---

### Workstream 4: Makefile & CI Integration

Update `Makefile` to include frontend UI tests in `make test`:
```makefile
test: generate
	@echo "--- Running Go Tests ---"
	@cd src/backend/api && go test -mod=vendor ./...
	@cd src/frontend/cli && go test -mod=vendor ./...
	@echo "--- Running Web UI Tests ---"
	@cd src/frontend/web && npm test
```

---

## 4. Future Work: End-to-End (E2E) Browser Testing with Playwright

To complement the in-memory test suite, full end-to-end browser testing using **Playwright** is planned as a future initiative.

### High-Level Scope (To be specced in a separate document):
- **Directory Location (`test/web`)**: All future browser tests will be located in the `test/web` directory at the root of the repository, keeping them alongside the sandbox API/CLI integration tests in `test/api`.
- **Real Browser Environment**: Running tests against real headless Chromium, WebKit, and Firefox instances.
- **True CSS Layout & Visual Checks**: Validating Tailwind layout responsiveness, sticky headers, z-index layers, and popover positioning.
- **Live Sandbox Integration**: Running tests against the live Docker Postgres + Go API backend sandbox (`make sandbox`).
- **Visual Regression Testing**: Screenshot comparison testing (`expect(page).toHaveScreenshot()`) for critical components and dark/light modes.
- **Separate Dedicated Target**: Configured under `make test-e2e` so daily unit testing and quick commits remain instant while full E2E tests can run in pre-release or CI pipelines.
