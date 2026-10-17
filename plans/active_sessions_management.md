# <a id="active-sessions-management-design" >Active Sessions Management Design</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Placement and Page Hierarchy](#placement-and-page-hierarchy)
  - [Compact Session Cards and Auto-Scrollable Container](#compact-session-cards-and-auto-scrollable-container)
  - [Revocation Workflows and Current Session Isolation](#revocation-workflows-and-current-session-isolation)
  - [Database Schema and IP/Location Tracking](#database-schema-and-ip-location-tracking)
  - [Protobuf API and ConnectRPC Service Contracts](#protobuf-api-and-connectrpc-service-contracts)
- [4. Assumptions](#4-assumptions)

---

## <a id="1-background" >1. Background</a>

In [plans/user_auth.md](user_auth.md), the core authentication architecture
established a hybrid token model: short-lived stateless JWT access tokens (1
hour TTL) paired with database-backed, revocable refresh tokens (30 days TTL
stored in the `refresh_tokens` table in PostgreSQL).

Section 4 ("Future work") of that plan identified the need for an **Active
Sessions Management UI**:

> Provide a self-service session management dashboard in user account settings
> allowing users to inspect active login sessions (IP address, approximate
> location, device hints) and revoke individual refresh tokens.

Currently, users can only log out from their current browser session or initiate
account deletion. If a user signs in from a borrowed device, public workstation,
or mobile phone, there is no self-service mechanism to inspect other concurrent
sessions or revoke access remotely without contacting support.

---

## <a id="2-goal" >2. Goal</a>

Implement a self-service **Active Sessions** management interface on the Account
Settings page
([`AccountSettingsPage.tsx`](../src/react/app-web/src/pages/AccountSettingsPage.tsx))
that allows authenticated users to inspect all active sessions across their
devices with a compact vertical footprint, automatically scrolls if there are
more than 3 sessions, clearly distinguishes the current device, and provides
both individual session revocation and bulk "Sign out all other sessions"
actions with confirmation dialogs.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:**
>
> - Extend `refresh_tokens` table with `ip_address` and `location_hint` columns.
> - Add `ListActiveSessions`, `RevokeSession`, and `RevokeOtherSessions` RPCs to
>   `AuthService`.
> - Display a dedicated "Active Sessions" card on the Account Settings page.
> - Render compact 2-row cards with device icons, browser/OS, IP, location, and
>   timestamps.
> - Enable smooth container scrolling when more than 3 sessions exist.

The proposed architecture connects the existing PostgreSQL `refresh_tokens`
storage directly to a self-service UI on the Account Settings page. When users
sign in via email OTP or magic link, the client's IP address and geographic
location hint are persisted alongside the User-Agent device hint. The frontend
queries active sessions via ConnectRPC, displays them in a compact, scrollable
list (highlighting the current device with an emerald badge), and lets users
revoke stale or unrecognized sessions immediately.

### <a id="placement-and-page-hierarchy" >Placement and Page Hierarchy</a>

The Active Sessions component will be placed on the Account Settings page
([`AccountSettingsPage.tsx`](../src/react/app-web/src/pages/AccountSettingsPage.tsx))
as a dedicated card section positioned immediately below the "Associated Email
Addresses" section and immediately above the "Public Profile" section.

This placement groups security and authentication credentials (email addresses,
login codes, active device sessions) logically together in the upper half of the
page, keeping cosmetic public profile customization (avatar, bio) and privacy
exports in dedicated sections below.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-tabbed-security-page" >Tabbed Security Page Navigation</a>

Split the Account Settings page into separate tabs (e.g., "Profile", "Security &
Sessions", "Data & Privacy").

**Pros:**

- Keeps individual page views slightly shorter by separating concerns.

**Cons:**

- Adds unnecessary tab navigation friction for a settings page with only 4
  concise sections.
- Breaks visual continuity with the existing single-scroll settings layout.

##### <a id="alt-bottom-placement" >Placement at Page Bottom Beside Data & Privacy</a>

Position the Active Sessions card at the very bottom of the page beneath Public
Profile and Data & Privacy.

**Pros:**

- Places active sessions near the bottom of the page next to other system
  actions.

**Cons:**

- Separates session security controls from associated email authentication
  controls.
- Forces users to scroll past public profile forms to inspect security events.

</details>

### <a id="compact-session-cards-and-auto-scrollable-container" >Compact Session Cards and Auto-Scrollable Container</a>

The Active Sessions component uses a space-efficient design that scales
gracefully as the user accumulates multiple device logins:

1. **Compact Vertical Height (`px-3.5 py-2.5`)**:
   - Each session card is condensed from ~90px down to ~54px tall.
   - Uses a tightly structured 2-row layout:
     - **Row 1 (Primary identity)**: Device & Browser name
       (`font-bold text-xs`), inline status badge (`● This Device` or device
       type tag), and relative activity timestamp (`• Active now` or
       `• Active 2h ago`).
     - **Row 2 (Network & provenance)**: IP address in monospace font,
       geographic location hint, and initial sign-in date separated by subtle
       bullet points.
2. **Auto-Scrollable Container (> 3 Sessions)**:
   - The session list is wrapped in an auto-scrolling container with
     `max-h-[224px] overflow-y-auto pr-1.5 space-y-2`.
   - Exactly 3 sessions fit within the container with the 4th session partially
     peeking below the fold to visually signal scrollability.
   - Styled with custom subtle scrollbar thumb rules
     (`rgba(161, 161, 170, 0.35)` in light theme, `rgba(82, 82, 91, 0.5)` in
     dark theme).
   - If 3 or fewer sessions exist, the container renders naturally without a
     scrollbar.
3. **Current Device Status Indicator**:
   - The current session is pinned to the top of the list and highlighted with
     an emerald border contour, soft emerald background fill, and a pulsing
     emerald badge (`● This Device`).

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-taller-unbounded-list" >Taller 3-Row Layout Without Height Limit</a>

Display each session as a tall 3-row card (~90px height) and let the container
expand indefinitely without a vertical height constraint.

**Pros:**

- Gives each piece of metadata (device, IP/location, creation date) its own
  dedicated row.

**Cons:**

- Pushes subsequent sections (Public Profile, Data & Privacy) far down the page
  when users have 5+ active devices.
- Clutters the account overview with excessive whitespace.

##### <a id="alt-compact-table-view" >Compact Tabular View</a>

Present active sessions as a dense HTML `<table>` with columns for Device,
Location, IP, Last Active, and Action.

**Pros:**

- Consumes less vertical space for users with a large number of sessions.

**Cons:**

- Degrades poorly on mobile viewports, requiring horizontal scrolling.
- Less visual hierarchy for distinguishing the current active device.

</details>

### <a id="revocation-workflows-and-current-session-isolation" >Revocation Workflows and Current Session Isolation</a>

The revocation workflow ensures users cannot accidentally lock themselves out
while providing decisive control over other devices:

1. **Current Session Isolation**:
   - The current device card does not have a "Revoke" button.
   - It is labeled with `"Current Session"` italic text and refers users to the
     primary "Log Out" action in the page header.
2. **Individual Session Revocation**:
   - Each remote session features a `"Revoke"` button (`px-2.5 py-1 text-xs`).
   - Clicking `"Revoke"` opens a confirmation modal detailing the device name
     and IP to confirm intentional revocation.
   - Upon confirmation, the backend deletes the corresponding `refresh_tokens`
     row.
3. **Bulk "Sign Out All Other Sessions"**:
   - The section header includes a `"Sign Out All Other Sessions"` button
     (enabled when at least one remote session exists).
   - Clicking opens a confirmation modal warning that all other devices will be
     immediately signed out.
   - The backend deletes all `refresh_tokens` for the user where
     `uuid != currentSessionUUID`.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-self-revocation-in-list" >Direct Current Session Revocation Inside List</a>

Include a "Revoke / Log Out" button directly on the current session card.

**Pros:**

- Allows revoking any session uniformly from the list.

**Cons:**

- Confusing UX when clicking "Revoke" causes immediate page unload and logout.
- Duplicates the explicit "Log Out" action already present in the page header.

##### <a id="alt-individual-only-revocation" >Individual-Only Revocation Without Bulk Action</a>

Provide only individual session revocation buttons without a bulk action.

**Pros:**

- Slightly simpler UI and fewer modal states.

**Cons:**

- Tedious for users with many stale mobile or desktop logins.
- Lacks a fast "panic button" if an account is suspected to be compromised.

</details>

### <a id="database-schema-and-ip-location-tracking" >Database Schema and IP/Location Tracking</a>

The `refresh_tokens` table in
[`src/db/quadsmith_api.sql`](../src/db/quadsmith_api.sql) is extended with two
columns:

```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
  uuid UUID NOT NULL,
  PRIMARY KEY (uuid),
  user_uuid UUID NOT NULL,
  token_hash TEXT UNIQUE NOT NULL,
  expires_at TIMESTAMPTZ,
  last_used_at TIMESTAMPTZ,
  device_hint TEXT NOT NULL,
  ip_address TEXT DEFAULT '' NOT NULL,
  location_hint TEXT DEFAULT '' NOT NULL,
  created_at TIMESTAMPTZ
);
```

[`src/db/quadsmith_api.mermaid`](../src/db/quadsmith_api.mermaid) is updated to
match.

In [`src/go/shared/pkg/auth/service.go`](../src/go/shared/pkg/auth/service.go),
`issueAuthSession` extracts `clientIP := getClientIP(req)` and
`locationHint := extractLocationHint(req.Header(), clientIP)` and inserts them
into `refresh_tokens`. In `RefreshToken`, `last_used_at` is updated to `NOW()`.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-omit-ip-location-schema" >Omit IP and Location from Schema</a>

Store only `device_hint` without extending the database schema.

**Pros:**

- Requires zero database schema alterations.

**Cons:**

- Prevents displaying IP and geographic location hints, degrading security
  transparency.

</details>

### <a id="protobuf-api-and-connectrpc-service-contracts" >Protobuf API and ConnectRPC Service Contracts</a>

The following RPCs are added to `AuthService` in
[`src/proto/quadsmith/api/auth.proto`](../src/proto/quadsmith/api/auth.proto):

```protobuf
service AuthService {
  // ... existing RPCs ...
  rpc ListActiveSessions(ListActiveSessionsRequest) returns (ListActiveSessionsResponse);
  rpc RevokeSession(RevokeSessionRequest) returns (RevokeSessionResponse);
  rpc RevokeOtherSessions(RevokeOtherSessionsRequest) returns (RevokeOtherSessionsResponse);
}

message Session {
  string uuid = 1;
  string device_hint = 2;
  string ip_address = 3;
  string location_hint = 4;
  google.protobuf.Timestamp created_at = 5;
  google.protobuf.Timestamp last_active_at = 6;
  bool is_current = 7;
}

message ListActiveSessionsRequest {}

message ListActiveSessionsResponse {
  repeated Session sessions = 1;
}

message RevokeSessionRequest {
  string session_uuid = 1;
}

message RevokeSessionResponse {
  bool success = 1;
  string message = 2;
}

message RevokeOtherSessionsRequest {}

message RevokeOtherSessionsResponse {
  bool success = 1;
  int32 revoked_count = 2;
  string message = 3;
}
```

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="alt-separate-session-service" >Dedicated SessionService Protocol</a>

Create a brand-new `SessionService` in a separate `session.proto` file.

**Pros:**

- Isolates session management in its own microservice definition.

**Cons:**

- Over-fragments the auth domain model when all session issuance, refresh, and
  revocation logic is already anchored in `AuthService`.

</details>

---

## <a id="4-assumptions" >4. Assumptions</a>

1. `cf-connecting-ip`, `x-forwarded-for`, and `x-real-ip` headers accurately
   reflect the client's public IP address in deployed and proxy environments.
2. The User-Agent string provides sufficient heuristics to classify the device
   type (Desktop, Mobile, Tablet, CLI) and identify the browser and OS family.
3. GeoIP lookup or proxy geo-headers (such as `cf-ipcity`, `cf-ipcountry`)
   supply the approximate location hint string when available.
