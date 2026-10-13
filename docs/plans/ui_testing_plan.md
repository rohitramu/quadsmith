# Web UI Testing Architecture Plan

## Table of Contents

- [1. Goal](#1-goal)
- [2. Proposal](#2-proposal)
  - [In-Memory Headless DOM Execution with Vitest and JSDOM](#in-memory-headless-dom-execution-with-vitest-and-jsdom)
  - [Type-Safe ConnectRPC In-Memory Mock Router Transports](#type-safe-connectrpc-in-memory-mock-router-transports)
  - [Application Decoupling for Isolated Route Navigation Testing](#application-decoupling-for-isolated-route-navigation-testing)
  - [Sub-Second Local and CI Integration Suite](#sub-second-local-and-ci-integration-suite)
- [3. Assumptions](#3-assumptions)
- [4. Future work](#4-future-work)
  - [End-to-End Browser Testing with Playwright](#end-to-end-browser-testing-with-playwright)

---

## 1. Goal

Establish a robust, hermetic, sub-second automated testing architecture for the Quadsmith web frontend ([`src/frontend/web`](../src/frontend/web)), verifying rendering integrity, interactive state machines, column customization, and multi-page client routing without requiring live backend servers or browser binaries.

---

## 2. Proposal

> **TL;DR:**
>
> - Execute component and routing tests headlessly in memory using [Vitest](https://vitest.dev/), [React Testing Library](https://testing-library.com/docs/react-testing-library/intro/), and [JSDOM](https://github.com/jsdom/jsdom).
> - Mock backend services at the protocol level using [ConnectRPC](https://connectrpc.com/)'s `createRouterTransport`, guaranteeing compile-time type safety against Protobuf definitions.
> - Decouple application route definitions from the browser DOM to enable multi-page navigation testing via `MemoryRouter`.
> - Embed UI tests directly into [`Makefile`](../Makefile)'s `make test` target, maintaining sub-second feedback loops in local development and continuous integration.

The web UI testing architecture prioritizes developer velocity, deterministic execution, and tight alignment with Quadsmith's Protobuf contracts. Rather than driving heavyweight browser instances for basic component verification, the frontend leverages an in-memory DOM simulation coupled with type-safe RPC mock transports. This guarantees that UI regressions, broken deep links, missing table columns, or invalid filter inputs are surfaced instantly.

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

### In-Memory Headless DOM Execution with Vitest and JSDOM

Component and navigation tests execute within Vitest using JSDOM as the virtual DOM environment. Tests render components using `@testing-library/react` and dispatch user interactions via `@testing-library/user-event`. This architecture verifies layout rendering, theme switching (dark/light mode toggles and `localStorage` persistence), sidebar navigation, catalog grids, and data table interactions in milliseconds.

#### Alternatives considered

Executing all unit, component, and routing tests inside real browser automation instances using tools like Playwright or Cypress.

**Pros:**

- Executes tests in genuine browser rendering engines with full layout, font rendering, and real Web APIs.

**Cons:**

- Introduces substantial startup latency (seconds vs. milliseconds), requiring browser binary management.
- Requires running live backend daemon processes or web servers, impeding rapid iteration during standard development workflows.

### Type-Safe ConnectRPC In-Memory Mock Router Transports

Rather than intercepting network requests with raw HTTP mocks or JSON fixture files, the test harness utilizes ConnectRPC's native `createRouterTransport`. Test fixtures are constructed as strongly-typed Protobuf TypeScript messages. The router transport routes client RPC queries (`ListMotors`, `GetMotor`, `ListFrames`) directly to in-memory TypeScript handlers, returning valid responses, pagination tokens, or error codes.

#### Alternatives considered

Using network-level mocking libraries such as Mock Service Worker (MSW) or mocking `fetch` calls directly.

**Pros:**

- Intercepts requests at the network layer and simulates low-level HTTP headers and connection drops.

**Cons:**

- Requires maintaining loose JSON fixture files that silently drift from generated Protobuf contracts when schemas change.
- Eliminates compile-time type safety across API boundaries.

### Application Decoupling for Isolated Route Navigation Testing

To test multi-step user navigation flows (such as navigating from the homepage to a hardware collection, clicking a component row, inspecting details, and following breadcrumbs back), application route hierarchies are decoupled from browser-dependent globals:

- **`AppRoutes` Export**: The core `<Routes>` hierarchy is extracted and exported independently of `<BrowserRouter>`.
- **`MemoryRouter` Harness**: Test utilities wrap routes inside React Router's `MemoryRouter`, allowing test assertions to simulate arbitrary initial paths (`/components/hardware/motors`), track navigation history, and verify breadcrumb transitions without a live URL bar.

### Sub-Second Local and CI Integration Suite

The test runner is configured to ensure the entire test suite executes in under two seconds. Test execution is wired directly into the root [`Makefile`](../Makefile) under `make test`:

```makefile
test: generate
	@echo "--- Running Go Tests ---"
	@cd src/backend/shared && go test -mod=vendor ./...
	@echo "--- Running Web UI Tests ---"
	@cd src/frontend/web && npm test
```

Any breaking UI change, failed assertion, or type mismatch blocks the build immediately.

---

## 3. Assumptions

1. Vitest and JSDOM provide sufficient DOM fidelity for React component interaction, event simulation, and client-side routing tests.
2. ConnectRPC's `createRouterTransport` faithfully models backend RPC handlers in memory without network traffic.
3. Tests run hermetically in local development and inside CI sandbox containers.

---

## 4. Future work

### End-to-End Browser Testing with Playwright

To complement the fast in-memory test suite, a dedicated end-to-end browser testing suite using Playwright is planned:

- **Sandbox Integration**: Tests will execute against the live Docker Postgres and Go backend sandbox (`make sandbox`).
- **Visual Regression Testing**: Screenshot comparisons (`toHaveScreenshot()`) to detect unintended visual styling or CSS regressions across viewport breakpoints and color themes.
- **Dedicated Target**: Isolated under `make test-e2e` to prevent slowing down daily commit workflows while ensuring thorough pre-release validation.
