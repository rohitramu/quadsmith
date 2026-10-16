# <a id="structured-bans-and-account-restrictions" >Structured Bans and Account Restrictions</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Dedicated Structured Bans Relational Model](#dedicated-structured-bans-relational-model)
  - [Persistent Banned Emails Across Account Deletion](#persistent-banned-emails-across-account-deletion)
  - [Fine-Grained Account Enforcement and Interceptor Controls](#fine-grained-account-enforcement-and-interceptor-controls)
  - [Administrative Moderation Protobuf Contracts in UserService](#administrative-moderation-protobuf-contracts-in-userservice)
  - [Dedicated Lock Screen and Account Settings Banned State](#dedicated-lock-screen-and-account-settings-banned-state)
  - [CLI Moderation Tooling in qs](#cli-moderation-tooling-in-qs)
  - [Post-Purge Username Hold Alignment](#post-purge-username-hold-alignment)
- [4. Alternatives considered](#4-alternatives-considered)
- [5. Assumptions](#5-assumptions)
- [6. Future work](#6-future-work)

---

## <a id="1-background" >1. Background</a>

In [user_auth.md](user_auth.md), Quadsmith introduced preliminary user account
states with a primitive `users.banned_at` timestamp. In that initial design,
banned accounts were conceptually hidden as `[Deleted User]` and permanently
barred from resurrection. However, the system lacked:

1. An audit trail identifying moderating administrators, infraction reasons, and
   ban durations.
2. Support for temporary suspensions with automated expiration.
3. Unban and revocation records explaining early reinstatement.
4. Granular access control distinguishing between suspended accounts and
   purged/deleted accounts.
5. Interactive UI states communicating suspension status directly to affected
   pilots while preserving personal identity management (display bio editing,
   email viewing, and session revocation).
6. A mechanism to keep all associated email addresses (primary and secondary)
   permanently banned even if a suspended user deletes their account.
   Furthermore, in [user_account_deletion.md](user_account_deletion.md),
   Quadsmith established the 30-day soft-deletion grace period and scrubber
   purge. During design alignment, we clarified that administrative bans are
   strictly reversible operational restrictions that must never trigger an
   automated data purge, and that vacated handles from deleted accounts follow a
   synchronized 30-day hold timer starting at `deleted_at`. This plan
   establishes a robust structured bans architecture in PostgreSQL, Go,
   Protobuf, CLI, and React, replacing `users.banned_at` with a normalized
   `bans` audit table and a persistent `banned_emails` registry.

---

## <a id="2-goal" >2. Goal</a>

Implement structured administrative moderation with temporary and permanent
bans, private moderator notes, revocation tracking, and persistent email bans
that survive account deletion. Banned users retain login access strictly to
their account page (where they can edit their display bio, inspect linked email
addresses, revoke individual or all active auth sessions, export GDPR data, and
delete their account), see a dedicated lock screen on other routes, and can be
banned or unbanned via `qs` CLI and `UserService` RPCs without modifying public
visibility to other community members.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:**
>
> - Replace `users.banned_at` with normalized `bans` and `banned_emails` tables
>   in PostgreSQL.
> - Persist bans on primary and secondary emails so deleted accounts cannot
>   re-register.
> - Banned users log in normally to manage settings (bio, sessions, emails,
>   privacy).
> - Other users observe no change in public content or profile availability.
> - Provide a dedicated lock screen, account settings alert, and `qs user ban`
>   CLI commands.

```mermaid
flowchart TD
    subgraph Admin Moderation
        Admin["Admin Pilot (qs user ban / UserService.BanUser)"] --> CreateBan["INSERT INTO bans"]
        CreateBan --> SnapshotEmails["INSERT INTO banned_emails (all user emails)"]
        AdminUnban["Admin Pilot (qs user bans update --revoke / UserService.UpdateBan)"] --> RevokeBan["UPDATE bans SET revoked_at = NOW(), revoked_by, revoke_reason"]
    end

    subgraph User Experience
        UserLogin["User Logs In / Authenticates"] --> CheckBannedEmail{"Email in active banned_emails?"}
        CheckBannedEmail -->|New Account Attempt| BlockRegistration["Reject: Email address suspended"]
        CheckBannedEmail -->|Existing User Account| AuthCheck{"Active Ban in bans?"}
        AuthCheck -->|No| NormalApp["Full Community Access"]
        AuthCheck -->|Yes| RestrictNav{"Requested Page / Procedure"}
        RestrictNav -->|Account Settings (Bio, Emails, Sessions, Privacy)| AllowAccess["Render AccountSettingsPage with Prominent Ban Alert"]
        RestrictNav -->|Other Pages / Non-Account RPCs| BlockAccess["Dedicated Lock Screen & PERMISSION_DENIED"]
    end

    subgraph Account Deletion Survival
        UserDelete["Banned User Deletes Account"] --> Scrubber["Purge Scrubber Runs (Day 30+)"]
        Scrubber --> WipeUser["Wipe user_emails, oauth_accounts, refresh_tokens"]
        WipeUser --> PersistBannedEmails["banned_emails records remain intact and active"]
    end
```

### <a id="dedicated-structured-bans-relational-model" >Dedicated Structured Bans Relational Model</a>

To enforce schema cleanliness and adhere to the zero-legacy rule, the primitive
`banned_at` column is removed from `users` in
[`src/db/quadsmith_api.sql`](../../src/db/quadsmith_api.sql) and
[`src/proto/quadsmith/api/user.proto`](../../src/proto/quadsmith/api/user.proto).
A dedicated `bans` table is introduced in `quadsmith_api`:

```sql
CREATE TABLE IF NOT EXISTS bans (
  uuid UUID NOT NULL,
  PRIMARY KEY (uuid),
  user_uuid UUID NOT NULL,
  banned_by_user_uuid UUID NOT NULL,
  reason TEXT NOT NULL,
  internal_notes TEXT DEFAULT '' NOT NULL,
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
  revoked_at TIMESTAMPTZ,
  revoked_by_user_uuid UUID,
  revoke_reason TEXT DEFAULT '' NOT NULL,
  FOREIGN KEY (user_uuid) REFERENCES users (uuid),
  FOREIGN KEY (banned_by_user_uuid) REFERENCES users (uuid),
  FOREIGN KEY (revoked_by_user_uuid) REFERENCES users (uuid)
);

CREATE INDEX IF NOT EXISTS idx_bans_user_uuid ON bans (user_uuid);
CREATE INDEX IF NOT EXISTS idx_bans_banned_by_user_uuid ON bans (banned_by_user_uuid);
CREATE INDEX IF NOT EXISTS idx_bans_created_at ON bans (created_at);
```

Key architectural rules:

- **Active Ban Check**: A user is actively banned if and only if there exists a
  row in `bans` where `user_uuid = $1 AND revoked_at IS NULL` and
  `(expires_at IS NULL OR expires_at > NOW())`.
- **Permanent Bans**: Represented cleanly by `expires_at IS NULL`.
- **Reversibility**: Permanent bans are fully reversible at any time by setting
  `revoked_at = NOW()`, preserving historical audit integrity.
- **Privacy Partitioning**: The user-facing `reason` is returned to the banned
  pilot, whereas `internal_notes` is strictly restricted to callers with
  `role = 'ADMIN'`.

#### <a id="alternatives-considered-database-schema" >Alternatives considered</a>

##### <a id="denormalized-banned-boolean-on-users" >Denormalized Banned Boolean on Users</a>

Retaining a `banned_at` or `is_banned` flag on the `users` table while writing
detailed logs to a separate moderation table. Fast queries inspect the flag
before querying details.

**Pros:**

- Avoids joining or querying the `bans` table for unbanned users during session
  checks.
- Simple single-row read on the `users` table.

**Cons:**

- Violates the single-source-of-truth principle by duplicating status across two
  tables.
- Introduces synchronization drift when temporary bans expire naturally without
  a database write.

##### <a id="hard-cascade-banning" >Hard Cascade Banning</a>

Anonymizing or deleting user profiles upon ban enactment, wiping active
credentials immediately.

**Pros:**

- Instantly removes offensive profiles from database storage.
- Requires no special routing or restricted states for banned accounts.

**Cons:**

- Destroys reversible audit data if a ban is successfully appealed.
- Disables self-service GDPR data export and account deletion rights for the
  pilot.

### <a id="persistent-banned-emails-across-account-deletion" >Persistent Banned Emails Across Account Deletion</a>

When a banned account is deleted (either during the 30-day soft-deletion grace
period or after the background scrubber executes final data anonymization), the
primary and additional email addresses must remain banned. This prevents a bad
actor from bypassing a ban simply by deleting their account and immediately
re-registering under the same email address. To support this without violating
referential integrity when `user_emails` rows are scrubbed, Quadsmith introduces
a dedicated `banned_emails` table:

```sql
CREATE TABLE IF NOT EXISTS banned_emails (
  uuid UUID NOT NULL,
  PRIMARY KEY (uuid),
  ban_uuid UUID NOT NULL,
  email TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
  FOREIGN KEY (ban_uuid) REFERENCES bans (uuid) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_banned_emails_email ON banned_emails (LOWER(email));
CREATE INDEX IF NOT EXISTS idx_banned_emails_ban_uuid ON banned_emails (ban_uuid);
```

Lifecycle behavior:

1. **Ban Enactment Snapshot**:
   - When an administrator bans a user via `UserService.BanUser` or
     `qs user ban`, the system queries all verified primary and secondary email
     addresses from `user_emails` for that `user_uuid`.
   - Each email address is normalized (lowercased, trimmed) and inserted into
     `banned_emails` linked to the active `ban_uuid`.
2. **Account Deletion & Purge Decoupling**:
   - If the banned user deletes their account, the 30-day scrubber executes
     `DELETE FROM user_emails WHERE user_uuid = $1` as required by GDPR.
   - Because `banned_emails` references `bans(uuid)` rather than
     `user_emails(uuid)`, the banned email records remain intact in
     `banned_emails`.
3. **Registration & OTP Interception**:
   - In `AuthService.SendEmailOtp`, before generating or dispatching an OTP
     challenge, the system checks whether the target email exists in
     `banned_emails` with an active parent ban:

     ```sql
     SELECT b.reason, b.expires_at
     FROM banned_emails be
     JOIN bans b ON b.uuid = be.ban_uuid
     WHERE LOWER(be.email) = LOWER($1)
       AND b.revoked_at IS NULL
       AND (b.expires_at IS NULL OR b.expires_at > NOW())
     ORDER BY b.created_at DESC
     LIMIT 1;
     ```

   - If an active ban is found and no existing active account owns the email
     (i.e. an attempt to create a new account with a banned email), the request
     is immediately rejected with `CodePermissionDenied` and message
     `"email address is suspended"`.
4. **Reversible Reinstatement**:
   - When an administrator revokes the ban (`UpdateBan` with `revoke = true`),
     setting `bans.revoked_at = NOW()` immediately deactivates the ban for all
     associated emails in `banned_emails`, restoring registration rights.

#### <a id="alternatives-considered-banned-emails" >Alternatives considered</a>

##### <a id="retaining-user-emails-rows-for-banned-users" >Retaining user_emails Rows for Banned Users</a>

Leaving `user_emails` rows in place during scrubber purge if the user has an
active ban record.

**Pros:**

- Avoids introducing a second table for email addresses.

**Cons:**

- Violates privacy compliance by continuing to link personal emails to deleted
  user records.
- Creates ambiguous orphan records in `user_emails` where `users` has been
  anonymized to `deleted-<uuid>`.

##### <a id="single-string-email-column-on-bans" >Single String Email Column on Bans</a>

Storing only the primary email address directly as a `TEXT` column on `bans`.

**Pros:**

- Simpler table schema with one column.

**Cons:**

- Drops all secondary/additional email addresses associated with the account,
  allowing the banned pilot to immediately re-register using any secondary
  address.

### <a id="fine-grained-account-enforcement-and-interceptor-controls" >Fine-Grained Account Enforcement and Interceptor Controls</a>

Banned users must still be able to log in, but should only be able to access
their account page and associated API methods. In accordance with user
specifications, suspended pilots are allowed to update their display bio,
inspect their linked emails, and manage their authentication sessions:

1. **Authentication Token Issuance**:
   - In
     [`src/go/shared/pkg/auth/service.go`](../../src/go/shared/pkg/auth/service.go),
     `issueAuthSession` checks for an active ban.
   - If an active ban exists, `AuthResponse` returns `account_status: "BANNED"`
     along with the active `Ban` record in `AuthResponse.active_ban`.
   - Access and refresh tokens are issued normally so the user can interact with
     permitted account endpoints.
2. **Auth Interceptor Enforcement**:
   - In
     [`src/go/shared/pkg/auth/interceptor.go`](../../src/go/shared/pkg/auth/interceptor.go),
     when a request is authenticated:
   - Check if the authenticated user has an active ban.
   - If banned, inspect the procedure name against an explicit allowlist:
     - `/quadsmith.api.AuthService/GetCurrentUser`
     - `/quadsmith.api.AuthService/RefreshToken`
     - `/quadsmith.api.AuthService/Logout`
     - `/quadsmith.api.AuthService/UpdateProfile` (permits updating display bio
       and avatar)
     - `/quadsmith.api.AuthService/ListUserEmails` (permits viewing linked
       emails)
     - `/quadsmith.api.AuthService/ListActiveSessions` (permits viewing
       sessions)
     - `/quadsmith.api.AuthService/RevokeSession` (permits revoking a single
       session)
     - `/quadsmith.api.AuthService/RevokeOtherSessions` (permits revoking all
       other sessions)
     - `/quadsmith.api.AuthService/SendEmailOtp`
     - `/quadsmith.api.AuthService/VerifyEmailOtp`
     - `/quadsmith.api.AuthService/DeleteAccount`
     - `/quadsmith.api.AuthService/CancelAccountDeletion`
     - `/quadsmith.api.UserService/GetUserBanStatus`
   - Any other procedure (such as `ChangeUsername`, `MergeAccount`,
     `AddComment`, `CreateBuild`, `RateBuild`) is rejected with Connect code
     `CodePermissionDenied` and error message
     `"account suspended: access restricted to account settings"`.
3. **Public Community Invariance**:
   - When public visitors or other pilots view the banned user's profile
     (`GetPublicProfile`), builds, or comments, the backend behaves identically
     to normal. Ban status is not leaked to third parties.

#### <a id="alternatives-considered-interceptor-enforcement" >Alternatives considered</a>

##### <a id="immediate-token-revocation-and-login-blocking" >Immediate Token Revocation and Login Blocking</a>

Revoking all refresh tokens immediately upon ban and rejecting all subsequent
login attempts with `CodePermissionDenied`.

**Pros:**

- Completely prevents banned actors from connecting to any API endpoint.
- Eliminates the need for per-procedure allowlist filtering in the interceptor.

**Cons:**

- Prevents the pilot from understanding why they were suspended.
- Blocks legal compliance features like GDPR self-service data export and
  voluntary account deletion.

### <a id="administrative-moderation-protobuf-contracts-in-userservice" >Administrative Moderation Protobuf Contracts in UserService</a>

The moderation RPCs are integrated directly into `UserService` in
[`src/proto/quadsmith/api/user.proto`](../../src/proto/quadsmith/api/user.proto),
with the SQL table definitions (`Ban` and `BannedEmail`) stored in
[`src/proto/quadsmith/store/user.proto`](../../src/proto/quadsmith/store/user.proto):

```protobuf
service UserService {
  rpc GetUser(GetUserRequest) returns (User);
  rpc GetPublicProfile(GetPublicProfileRequest) returns (PublicUserProfile);
  rpc CheckUsernameAvailable(CheckUsernameAvailableRequest) returns (CheckUsernameAvailableResponse);

  // Administrative Moderation RPCs (guarded by ADMIN role)
  rpc BanUser(BanUserRequest) returns (BanUserResponse);
  rpc UpdateBan(UpdateBanRequest) returns (UpdateBanResponse);
  rpc ListBans(ListBansRequest) returns (ListBansResponse);
  rpc GetUserBanStatus(GetUserBanStatusRequest) returns (GetUserBanStatusResponse);
}

// In src/proto/quadsmith/store/user.proto:
message Ban {
  option (quadsmith.sql.table) = {name: "bans"};
  option (quadsmith.data.export) = {
    domain: EXPORT_DOMAIN_SECURITY
    user_filter: "user_uuid = $1::uuid"
  };

  string uuid = 1 [
    (quadsmith.sql.column) = {is_primary_key: true},
    (quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC
  ];
  string user_uuid = 2 [
    (quadsmith.sql.column) = {
      references: "users(uuid)"
      create_index: true
    },
    (quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC
  ];
  string banned_by_user_uuid = 3 [
    (quadsmith.sql.column) = {
      references: "users(uuid)"
      create_index: true
    },
    (quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC
  ];
  string reason = 4 [(quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC];
  string internal_notes = 5 [(quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PRIVATE];
  google.protobuf.Timestamp expires_at = 6 [(quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC];
  google.protobuf.Timestamp created_at = 7 [
    (quadsmith.sql.column) = {default_value: "NOW()"},
    (quadsmith.data.content_type) = CONTENT_TYPE_USER_OBSERVED
  ];
  google.protobuf.Timestamp revoked_at = 8 [(quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC];
  optional string revoked_by_user_uuid = 9 [
    (quadsmith.sql.column) = {references: "users(uuid)"},
    (quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC
  ];
  string revoke_reason = 10 [(quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC];
}

message BannedEmail {
  option (quadsmith.sql.table) = {name: "banned_emails"};
  option (quadsmith.data.export) = {
    domain: EXPORT_DOMAIN_SECURITY
    user_filter: "ban_uuid IN (SELECT uuid FROM bans WHERE user_uuid = $1::uuid)"
  };

  string uuid = 1 [
    (quadsmith.sql.column) = {is_primary_key: true},
    (quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC
  ];
  string ban_uuid = 2 [
    (quadsmith.sql.column) = {
      references: "bans(uuid) ON DELETE CASCADE"
      create_index: true
    },
    (quadsmith.data.content_type) = CONTENT_TYPE_SYSTEM_METADATA_PUBLIC
  ];
  string email = 3 [
    (quadsmith.sql.column) = {create_index: true},
    (quadsmith.data.content_type) = CONTENT_TYPE_USER_PROVIDED_PRIVATE
  ];
  google.protobuf.Timestamp created_at = 4 [
    (quadsmith.sql.column) = {default_value: "NOW()"},
    (quadsmith.data.content_type) = CONTENT_TYPE_USER_OBSERVED
  ];
}

message BanUserRequest {
  string user_id = 1; // Username or UUID of user to ban
  string reason = 2; // Public reason visible to banned user
  string internal_notes = 3; // Private admin moderation notes
  optional google.protobuf.Timestamp expires_at = 4; // Nullable for permanent bans
}

message BanUserResponse {
  bool success = 1;
  string message = 2;
  Ban ban = 3;
}

message UpdateBanRequest {
  string user_id = 1; // Username or UUID of user to update
  optional string reason = 2; // Updated public infraction reason
  optional string internal_notes = 3; // Updated private staff notes
  optional google.protobuf.Timestamp expires_at = 4; // Updated expiration timestamp
  bool clear_expires_at = 5; // Set true to convert temporary ban into permanent ban
  optional bool revoke = 6; // Set true to lift/revoke the ban immediately
  optional string revoke_reason = 7; // Moderator reason for lifting/revoking the ban
}

message UpdateBanResponse {
  bool success = 1;
  string message = 2;
  Ban ban = 3;
}

message ListBansRequest {
  optional string user_id = 1; // Filter by user (optional)
  int32 page_size = 2;
  string page_token = 3;
  string filter = 4;
  repeated string sort = 5;
  repeated string columns = 6;
}

message ListBansResponse {
  repeated Ban bans = 1;
  string next_page_token = 2;
}

message GetUserBanStatusRequest {
  string user_id = 1; // Username or UUID (can inspect self, or any user if ADMIN)
}

message GetUserBanStatusResponse {
  bool is_banned = 1;
  optional Ban active_ban = 2;
}
```

#### <a id="alternatives-considered-protobuf-services" >Alternatives considered</a>

##### <a id="separate-moderationservice-package" >Separate ModerationService Package</a>

Placing ban and unban procedures in a dedicated `ModerationService` in
`proto/quadsmith/api/moderation.proto`.

**Pros:**

- Isolates moderation workflows into a dedicated API contract.
- Keeps `UserService` focused strictly on profile reads.

**Cons:**

- Proliferates micro-services and Connect handlers for a small set of methods.
- Requires configuring and deploying an additional RPC route in server routing.

### <a id="dedicated-lock-screen-and-account-settings-banned-state" >Dedicated Lock Screen and Account Settings Banned State</a>

The web frontend in [`src/react/app-web`](../../src/react/app-web) implements
two dedicated components for suspended accounts:

1. **Dedicated Lock Screen (`AccountRestrictedPage.tsx`)**:
   - When an authenticated user with an active ban navigates to any non-account
     route (`/`, `/builds`, `/build/new`, `/parts`, `/collections`), the layout
     or router middleware displays the dedicated `AccountRestrictedPage`.
   - Displays a prominent shield icon with crimson/amber styling, indicating
     "Account Access Restricted".
   - Shows the specific reason for suspension under the concise header
     **"Reason"**, the date issued, and either the expiration date or
     "Indefinite suspension".
   - Provides a prominent direct action button: `[Account Settings]`.
2. **Account Settings Ban State (`AccountSettingsPage.tsx`)**:
   - When on `/settings`, the page displays a prominent alert banner at the very
     top of the page detailing the active suspension terms.
   - **Profile Section**: The username handle is locked (read-only), but the
     **display bio is fully editable** so the pilot can update their personal
     description.
   - **Linked Emails Section**: Displays all verified primary and secondary
     email addresses in read-only mode so the user can verify their connected
     identities (with add/merge actions disabled during suspension).
   - **Auth Sessions Section**: Displays all active login sessions with device
     and IP hints, featuring an active **"Revoke All Other Sessions"** button
     and per-session revoke controls.
   - **Privacy Section**: Features self-service data export and voluntary
     account deletion under the concise heading **"Privacy"**.

```text
+--------------------------------------------------------------------------+
|  [!] Account Access Restricted                                           |
|  Your account was suspended on Oct 14, 2026.                             |
|  Reason: Violation of community flight safety guidelines.                |
|  Duration: Indefinite suspension                                         |
+--------------------------------------------------------------------------+
|  Profile Information (Handle: Locked, Bio: Editable)                     |
|  ...                                                                     |
|  Linked Emails (Read-Only: View Verified Addresses)                      |
|  ...                                                                     |
|  Auth Sessions (Active - Revoke All Sessions & Manage)                   |
|  ...                                                                     |
|  Privacy (Active)                                                        |
|  [Export My Data]   [Delete Account]                                     |
+--------------------------------------------------------------------------+
```

#### <a id="alternatives-considered-frontend-presentation" >Alternatives considered</a>

##### <a id="toast-notification-with-silent-redirect" >Toast Notification with Silent Redirect</a>

Instantly redirecting banned users to `/settings` whenever they attempt to visit
another route, displaying a brief temporary toast notification.

**Pros:**

- Avoids rendering a dedicated full-page lock screen component.
- Keeps users directly on the single page they are permitted to use.

**Cons:**

- Confusing UX as navigation links appear broken or glitchy to the user.
- Fails to clearly explain why navigation was intercepted.

### <a id="cli-moderation-tooling-in-qs" >CLI Moderation Tooling in qs</a>

Administrative pilots managing Quadsmith use the `qs` CLI in
[`src/go/cli`](../../src/go/cli) to enforce moderation actions following the
standard entity collection interface
`qs user bans [get|list|new|update] <username|uuid>` where the target identifier
consistently follows the operation name:

1. **New Ban**:

   ```bash
   qs user bans new <username|uuid> --reason "Spamming commercial links" [--expires-in 7d|30d] [--notes "Reported by multiple pilots"]
   # (alias: qs user bans create <username|uuid> ...)
   ```

2. **Get Active Ban Details**:

   ```bash
   qs user bans get <username|uuid> [-c column,...] [--yaml] [--json]
   ```

3. **List Ban History**:

   ```bash
   qs user bans list [username|uuid] [-f filter] [-s sort,...] [-c column,...] [-l 50] [--yaml] [--json]
   ```

4. **Update Active Ban**:

   ```bash
   qs user bans update <username|uuid> [--reason "..."] [--expires-in 14d] [--permanent] [--revoke] [--notes "..."]
   ```

Because the `bans` collection serves as an immutable moderation audit log, it
strictly allows `new` (with `create` alias) and `update` mutation operations
(omitting `delete`). Active bans are modified or revoked via `update`, and the
`list` operation supports standard sorting, column projection, CEL filtering,
pagination limits, and output formatting (`--json` and `--yaml`) flags common to
all Quadsmith entity collections. All commands require `Role == "ADMIN"`.

#### <a id="alternatives-considered-cli-placement" >Alternatives considered</a>

##### <a id="delete-mutation-for-unbanning" >Delete Mutation for Unbanning</a>

Treating unbans as a `delete` operation on the `bans` collection (e.g.,
`qs user bans delete <username|uuid>`).

**Pros:**

- Conforms to conventional CRUD verb sets where deletion inverts creation.

**Cons:**

- Misrepresents the underlying data model by implying ban records are wiped from
  database history.
- The `bans` collection functions as an append-only audit trail and moderation
  log where active bans are either updated (shortened, lengthened, made
  permanent) or revoked rather than purged.

##### <a id="ops-cli-only-enforcement" >Ops CLI Only Enforcement</a>

Implementing ban commands exclusively in `qs-ops` using internal operator tokens
rather than `qs`.

**Pros:**

- Keeps user-facing pilot CLI completely free of moderation logic.
- Requires operator token authentication through ops server.

**Cons:**

- Prevents community moderators who authenticate through web/CLI credentials
  from moderating accounts.
- Inconvenient for developer workflows and quick testing in the local sandbox.

### <a id="post-purge-username-hold-alignment" >Post-Purge Username Hold Alignment</a>

In alignment with
[`docs/plans/user_account_deletion.md`](user_account_deletion.md):

1. **Single Source of Truth**:
   - The deletion timer and username hold timer start at the exact same
     timestamp: `users.deleted_at`.
   - During the 30-day grace period, the handle is held natively in `users.id`
     via PostgreSQL unique constraints.
2. **30-Day Purge Scrubber**:
   - When the 30-day grace period expires, `HandlePurgeSoftDeletedAccounts`
     renames `users.id` to `deleted-<uuid>` and wipes private identifiers.
   - The original username handle is released cleanly at that same 30-day mark.
3. **Separation from Bans**:
   - Administrative bans are access restrictions, not deletion triggers.
   - Bans never initiate account purges, ensuring permanent bans remain fully
     reversible by administrators without data loss.

#### <a id="alternatives-considered-purge-alignment" >Alternatives considered</a>

##### <a id="additional-30-day-post-purge-cooldown-in-username-histories" >Additional 30-Day Post-Purge Cooldown in username_histories</a>

Adding an extra 30-day cooldown starting at purge time (day 30 to day 60) by
inserting into `username_histories`.

**Pros:**

- Adds further protection against immediate handle sniping of prominent deleted
  accounts.

**Cons:**

- Extends total handle lockup to 60 days, delaying legitimate re-registration.
- Disaligns the purge execution from handle release.

---

## <a id="4-alternatives-considered" >4. Alternatives considered</a>

### <a id="temporary-shadow-banning" >Temporary Shadow Banning</a>

Silently hiding all user posts and builds from other users while letting the
banned user believe everything is functioning normally.

**Pros:**

- Slows down spammers who do not immediately realize their content is hidden.

**Cons:**

- Deceptive and violates Quadsmith's open engineering principles.
- Creates severe debugging and customer support overhead.

---

## <a id="5-assumptions" >5. Assumptions</a>

1. Only accounts with `role = 'ADMIN'` are authorized to invoke `BanUser`,
   `UpdateBan`, and `ListBans`.
2. Banned accounts retain verified email addresses and OAuth associations to
   allow authentication and unbanning.
3. Public communal content created prior to the ban remains visible without
   modification, preserving community flight schematics and troubleshooting
   records.
4. Permanent bans do not expire automatically and require explicit
   administrative revocation.

---

## <a id="6-future-work" >6. Future work</a>

### <a id="moderation-audit-log-webhook-dispatch" >Moderation Audit Log Webhook Dispatch</a>

Dispatch administrative moderation events (bans, revocations, warnings) to an
internal audit logging webhook or Slack/Discord channel for real-time visibility
across the moderator team.

### <a id="automated-appeal-submission-workflow" >Automated Appeal Submission Workflow</a>

Provide a structured appeal form within the `AccountRestrictedPage` allowing
suspended pilots to submit a formal ban appeal with supporting context directly
to administrators.
