# <a id="protobuf-top-down-structure-and-organization" >Protobuf Top-Down Structure and Organization</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Deterministic Four-Section Protobuf File Structure](#four-section-structure)
  - [Message Reference Ordering](#message-reference-ordering)
  - [Modular Per-Binary Configuration Decomposition](#modular-per-binary-configuration-decomposition)
  - [Codified AGENTS.md Rule Specification](#codified-agents-md-rule-specification)
- [4. Assumptions](#4-assumptions)

---

## <a id="1-background" >1. Background</a>

Protobuf definitions across the repository previously exhibited inconsistent
internal organization. In schemas such as configuration and authentication,
primitives, enums, and low-level leaf messages were placed at the very top of
the file, forcing engineers and automated tools to scroll past low-level schemas
before discovering the primary `service` declaration or root configuration
messages.

The user clarified the architectural conventions:

> _"Proto files should be structured like this:_ _1. Services_ _2. Domain model_
> _3. Request/response messages_ _4. Everything else_
>
> _Remove the numbering of these sections. Remove the explanation of ordering_
> _(currently in parentheses after the section name)._
>
> _Sort messages based on what references them - a message should never be
> above_ _what references them."_

This document formalizes these rules into a permanent engineering standard.

---

## <a id="2-goal" >2. Goal</a>

Establish a deterministic structural ordering standard for all Protobuf
definitions (`.proto`) in [AGENTS.md](../AGENTS.md) that organizes files into
four standard sections (Services, Domain model, Request/response messages,
Everything else) with clean unnumbered headers, mandates that a message is never
above what references it, applies alphabetical sorting as the lowest-precedence
tie-breaker among peer definitions at the same tier, and organizes application
configurations into modular per-binary proto files.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:**
>
> - Organize Protobuf definitions into Services, Domain model, Request/response
>   messages, and Everything else.
> - Use clean unnumbered section headers without parenthetical ordering
>   explanations.
> - Sort messages based on what references them so a message is never above what
>   references it.
> - Sort peer definitions at the same tier alphabetically as a lowest-precedence
>   tie-breaker.
> - Decompose application configurations into per-binary proto files sharing a
>   common package.

Protobuf files follow a deterministic structure where readers encounter the
high-level service interface and domain models first, followed by specific
request and response payloads, and finally shared supporting types, enums, and
extensions.

### <a id="four-section-structure" >Deterministic Four-Section Protobuf File Structure</a>

Every `.proto` file in the repository must be arranged in the following order
(omitting any section that has no definitions in that file):

1. `// Services`: RPC `service` interface definitions.
2. `// Domain model`: Primary entity models, persistence tables, and domain
   objects.
3. `// Request/response messages`: RPC request and response payload messages.
4. `// Everything else`: Shared supporting types, nested data records, enums,
   and custom protobuf option extensions.

Section headers use clean, unnumbered comment blocks without parenthetical
ordering explanations:

```proto
// ---------------------------------------------------------
// Services
// ---------------------------------------------------------

// ---------------------------------------------------------
// Domain model
// ---------------------------------------------------------

// ---------------------------------------------------------
// Request/response messages
// ---------------------------------------------------------

// ---------------------------------------------------------
// Everything else
// ---------------------------------------------------------
```

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="numbered-section-headers" >Numbered Section Headers with Ordering Explanations</a>

Retain numbers (`// 1.`, `// 2.`) and parenthetical ordering explanations (such
as `(Services First)` or `(Callers Precede Callees)`) in section comment blocks.

**Pros:**

- Explicitly announces the underlying rationale directly in comments.

**Cons:**

- Visual clutter that distracts from reading schema code.
- Becomes brittle and repetitive across dozens of schema files.

##### <a id="bottom-up-ordering" >Bottom-Up Ordering (C/C++ Header Style)</a>

Require dependencies to be declared before they are referenced, mimicking C/C++
forward-declaration requirements.

**Pros:**

- Familiar to developers accustomed to single-pass compilation languages.

**Cons:**

- Obscures the most important entrypoint or service contract at the bottom of
  large schema files.
- Contradicts the user's explicit mandate for top-down architecture.

</details>

### <a id="message-reference-ordering" >Message Reference Ordering</a>

Sort messages based on what references them — a message should never be above
what references it:

1. **Referenced Below Referencer**: When definition $A$ has a field of type $B$
   ($A$ references $B$), definition $B$ must appear below definition $A$.
2. **Never Above Referencers**: A message or enum must never be placed above any
   message that references it.
3. **Alphabetical Tie-Breaker**: When multiple messages or definitions exist at
   the same structural tier without mutual references, sort them in alphabetical
   order. This operates strictly as a lowest-precedence tie-breaker.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="strict-global-alphabetical-ordering" >Strict Global Alphabetical Ordering</a>

Impose pure alphabetical sorting across all definitions regardless of reference
relationships.

**Pros:**

- Completely deterministic with simple linter automation.

**Cons:**

- Scatters referencing messages and their children across the entire file.
- Violates the rule that a message should never be above what references it.

</details>

### <a id="modular-per-binary-configuration-decomposition" >Modular Per-Binary Configuration Decomposition</a>

Rather than maintaining a monolithic `config.proto` file that conflates server,
ops, job, web, and CLI configuration schemas, split definitions into dedicated
per-binary schema files under `src/proto/quadsmith/config/`:

- `common.proto`: Shared infrastructure (`DatabaseConfig`, `ServerConfig`,
  `BootstrapTokenConfig`, `InternalGroupConfig`).
- `server_api.proto`: `ApiServerConfig`, `AuthConfig`, `EmailConfig`,
  `ConsoleEmailConfig`, `SmtpConfig`.
- `server_ops_api.proto`: `OpsApiServerConfig`, `JitPolicyConfig`,
  `OAuthConfig`, `DeveloperMapping`.
- `server_ops_web.proto`: `OpsWebServerConfig`.
- `server_cron.proto`: `CronServerConfig`, `CleanupExpiredOtpsSchedule`,
  `CleanupExpiredTokensSchedule`, `CleanupOrphanedBlobsSchedule`,
  `PurgeSoftDeletedAccountsSchedule`, `SendUsernameReleaseWarningsSchedule`.
- `server_web.proto`: `WebServerConfig`.
- `cli.proto`: `CliConfig`.
- `cli_ops.proto`: `OpsCliConfig`.

All files share `package quadsmith.config;` and target the same Go package
(`quadsmith/shared/gen/quadsmith/config;config`), preserving complete
source-level compatibility across all existing Go and TypeScript call sites.

<details>
<summary><strong>Alternatives considered</strong></summary>

##### <a id="monolithic-single-config-file" >Monolithic Single Config File</a>

Retain all configuration messages in a single `config.proto` file.

**Pros:**

- Requires only one file in `src/proto/quadsmith/config/`.

**Cons:**

- Forces unrelated binaries to share a single sprawling schema file.
- Conflates top-down ordering across distinct application domains.

</details>

### <a id="codified-agents-md-rule-specification" >Codified AGENTS.md Rule Specification</a>

The complete rule in [AGENTS.md](../AGENTS.md) under
`## Protobuf Structure & Organization`:

```markdown
## Protobuf Structure & Organization

- **Proto File Section Structure**: All Protobuf files (`.proto`) must be
  structured with the following section order (omitting any section that has no
  definitions in that file):
  1. `// Services`
  2. `// Domain model`
  3. `// Request/response messages`
  4. `// Everything else`
  - Never number section headers (e.g., do not write `// 1. Services`), and
    never include parenthetical ordering explanations (e.g., do not write
    `(Services First)`).
- **Message Reference Ordering**: Sort messages based on what references them —
  a message should never be above what references them.
  - When a message references another definition, the referenced definition must
    appear below the referencing definition.
  - When multiple definitions exist at the same tier without references between
    them, sort them in alphabetical order as a lowest-precedence tie-breaker.
```

---

## <a id="4-assumptions" >4. Assumptions</a>

1. The Protobuf compiler (`protoc` / `buf`) supports forward references within
   the same `.proto` file, allowing callers to precede callees without
   compilation errors.
2. The reference ordering rule always takes precedence over alphabetical sorting
   whenever a conflict arises.
3. Peer messages within the same section that do not reference each other can be
   sorted alphabetically without altering wire-format compatibility or Protobuf
   field numbering semantics.
