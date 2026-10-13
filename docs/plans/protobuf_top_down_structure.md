# <a id="protobuf-top-down-structure-and-organization" >Protobuf Top-Down Structure and Organization</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Top-Down Structural Invariant for All Protobuf Definitions](#top-down-structural-invariant)
  - [Alphabetical Ordering as Lowest-Precedence Tie-Breaker](#alphabetical-ordering-tie-breaker)
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

The user clarified the desired architectural convention:

> _"I want the proto files in this repo to have a "top-down" structure. A
> message that gets used by messages should be below all the messages that use
> it. This not only applies to messages, but to all proto definitions (enums,
> services, etc.)."_

Subsequently, two refinements were approved:

1. Apply alphabetical sorting to peer definitions at the same hierarchy tier as
   a lowest-precedence tie-breaker.
2. Decompose monolithic configuration schemas into dedicated per-binary `.proto`
   files within `proto/quadsmith/config/`.

This document formalizes these rules into a permanent engineering standard.

---

## <a id="2-goal" >2. Goal</a>

Establish a deterministic structural ordering standard for all Protobuf
definitions (`.proto`) in [AGENTS.md](../../AGENTS.md) that mandates top-down
layout (callers/parents precede callees/children) as the primary invariant,
applies alphabetical sorting as the lowest-precedence tie-breaker among peer
definitions at the same tier, and organizes application configurations into
modular per-binary proto files.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:**
>
> - Organize Protobuf definitions top-down so callers and parents precede
>   callees and children.
> - Position shared dependencies, leaf messages, and enums below all referencing
>   definitions.
> - Sort independent definitions at the same tier alphabetically as a
>   lowest-precedence tie-breaker.
> - Decompose application configurations into per-binary proto files sharing a
>   common package.
> - Codify this deterministic ordering standard in AGENTS.md across the
>   codebase.

Protobuf files follow a top-down hierarchy where readers begin at the highest
level of conceptual abstraction (services and root messages) and proceed
downward through increasingly granular components and leaf records. Whenever
multiple definitions occupy the same hierarchy tier without dependency
constraints between them, they are arranged in alphabetical order.

### <a id="top-down-structural-invariant" >Top-Down Structural Invariant for All Protobuf Definitions</a>

Every `.proto` file in the repository must be arranged from highest level of
abstraction (callers, parents, entrypoints) to lowest level of abstraction
(callees, children, leaves):

1. **Services First**: If a file defines a `service`, the `service` block must
   appear at the very top of the file, preceding the request and response
   messages that it invokes.
2. **Parents Above Children**: A composite message must appear above the
   sub-messages it references.
3. **Usage Dictates Position**: If definition $B$ is referenced by definition
   $A$, $B$ must appear below $A$.
4. **Shared Dependencies Below All Usages**: If definition $B$ is referenced by
   multiple definitions $\{A_1, A_2, \dots, A_n\}$, $B$ must appear below the
   last of those definitions ($\max(\text{line}(A_i))$).
5. **Enums and Leaf Types Last**: Shared enums and primitive leaf records must
   appear below all messages that reference them.

#### <a id="alternatives-top-down" >Alternatives considered</a>

##### <a id="bottom-up-ordering" >Bottom-Up Ordering (C/C++ Header Style)</a>

Require dependencies to be declared before they are referenced, mimicking C/C++
forward-declaration requirements. This convention was historically common in
older programming toolchains.

**Pros:**

- Familiar to developers accustomed to single-pass compilation languages.

**Cons:**

- Obscures the most important entrypoint or service contract at the bottom of
  large schema files.
- Contradicts the user's explicit mandate for top-down architecture.

##### <a id="strict-one-definition-per-file" >Strict One-Definition-Per-File Decomposition</a>

Decompose all services, messages, and enums into individual single-item `.proto`
files, eliminating intra-file ordering ambiguity altogether.

**Pros:**

- Guarantees zero ambiguity regarding message order within any given file.

**Cons:**

- Fractures the repository schema into dozens of tiny single-definition files.
- Introduces substantial import graph overhead and complicates code generation.

### <a id="alphabetical-ordering-tie-breaker" >Alphabetical Ordering as Lowest-Precedence Tie-Breaker</a>

When multiple messages or definitions exist at the same tier or hierarchy level
without mutual dependency constraints, sort them in alphabetical order. This
guideline operates strictly as a tie-breaker and takes lowest precedence:

1. **Tier Identification**: Definitions that share identical dependency depths
   form an equivalence tier.
2. **Alphabetical Tie-Break**: Within that tier, sort names alphabetically.
3. **Top-Down Priority**: If definition $X$ alphabetically precedes definition
   $Y$, but $X$ is referenced by $Y$, the top-down structural invariant strictly
   overrides alphabetical sorting ($Y$ must precede $X$).

#### <a id="alternatives-alphabetical" >Alternatives considered</a>

##### <a id="strict-alphabetical-ordering" >Strict Global Alphabetical Ordering</a>

Impose pure alphabetical sorting across all definitions regardless of
caller/callee or parent/child relationships. This eliminates any tier judgment
calls and simplifies automated lint rules.

**Pros:**

- Completely deterministic with trivial linter automation.

**Cons:**

- Scatters parent messages and their immediate children across the entire file.
- Forces services and primary entrypoint messages into arbitrary alphabetical
  positions, destroying high-level readability.

##### <a id="arbitrary-tier-ordering" >Arbitrary or Freeform Ordering Within Tiers</a>

Allow definitions at the same tier to be ordered arbitrarily by author
discretion without alphabetical sorting. This imposes fewer formatting
constraints during initial schema drafting.

**Pros:**

- Authors can group loosely related sibling messages without strict
  alphabetization.

**Cons:**

- Leads to subjective placement and frequent churn across pair-programming
  edits.
- Fails to provide a clear, deterministic tie-breaker for peer definitions.

### <a id="modular-per-binary-configuration-decomposition" >Modular Per-Binary Configuration Decomposition</a>

Rather than maintaining a monolithic `config.proto` file that conflates server,
ops, job, web, and CLI configuration schemas, split definitions into dedicated
per-binary schema files under `proto/quadsmith/config/`:

- `common.proto`: Shared infrastructure (`DatabaseConfig`, `ServerConfig`).
- `server_api.proto`: `ApiServerConfig`, `AuthConfig`, `EmailConfig`,
  `ConsoleEmailConfig`, `SmtpConfig`.
- `server_ops.proto`: `OpsServerConfig`, `BootstrapTokenConfig`,
  `InternalGroupConfig`, `JitPolicyConfig`, `OAuthConfig`, `DeveloperMapping`.
- `server_jobs.proto`: `JobServerConfig`, `JobScheduleConfig`.
- `server_web.proto`: `WebServerConfig`.
- `cli.proto`: `CliConfig`.
- `cli_ops.proto`: `OpsCliConfig`.

All files share `package quadsmith.config;` and target the same Go package
(`quadsmith/shared/gen/quadsmith/config;config`), preserving complete
source-level compatibility across all existing Go and TypeScript call sites.

#### <a id="alternatives-decomposition" >Alternatives considered</a>

##### <a id="monolithic-single-config-file" >Monolithic Single Config File</a>

Retain all configuration messages in a single `config.proto` file.

**Pros:**

- Requires only one file in `proto/quadsmith/config/`.

**Cons:**

- Forces unrelated binaries to share a single sprawling schema file.
- Conflates top-down ordering across distinct application domains.

### <a id="codified-agents-md-rule-specification" >Codified AGENTS.md Rule Specification</a>

The complete rule in [AGENTS.md](../../AGENTS.md) under
`## Protobuf Structure & Organization`:

```markdown
## Protobuf Structure & Organization

- **Top-Down Structural Invariant**: All Protobuf files (`.proto`) must be
  organized strictly "top-down". Any definition (message, enum, service, etc.)
  that is referenced or used by other definitions must appear below all the
  definitions that use it.
  - **Services First**: `service` definitions must appear at the top of the
    file, preceding the request and response messages they call.
  - **Callers / Parents Precede Callees / Children**: A message that embeds or
    references other messages must be placed above the messages it references.
  - **Shared Dependencies Below All Usages**: If a message or enum is referenced
    by multiple messages, it must appear below all messages that reference it.
  - **Enums & Leaf Types**: Enums and shared leaf records must appear below all
    messages that reference them.
  - **Alphabetical Ordering (Lowest Precedence)**: When multiple definitions
    exist at the same structural tier without dependency constraints between
    them, sort them in alphabetical order. This rule takes lowest precedence and
    must never violate the top-down structural invariant.
```

---

## <a id="4-assumptions" >4. Assumptions</a>

1. The Protobuf compiler (`protoc` / `buf`) supports forward references within
   the same `.proto` file, allowing callers to precede callees without
   compilation errors.
2. The top-down invariant always takes precedence over alphabetical sorting
   whenever a conflict arises.
3. Peer messages within the same hierarchy tier that do not reference each other
   can be sorted alphabetically without altering wire-format compatibility or
   Protobuf field numbering semantics.
