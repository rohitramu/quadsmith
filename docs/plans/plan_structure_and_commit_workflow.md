# Standardized Plan Structure and Commit Workflow

## Table of Contents
- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Design Decision 1: Plan Document Structure & Section Rules](#design-decision-1-plan-document-structure--section-rules)
  - [Design Decision 2: Plan Lifecycle (Artifacts to docs/plans/)](#design-decision-2-plan-lifecycle-artifacts-to-docsplans)
  - [Design Decision 3: Git Commit Workflow Integration](#design-decision-3-git-commit-workflow-integration)
  - [Design Decision 4: Plan Maintenance, Superseding, and Archival](#design-decision-4-plan-maintenance-superseding-and-archival)
  - [Design Decision 5: Self-Execution for this Change](#design-decision-5-self-execution-for-this-change)
- [4. Alternatives considered](#4-alternatives-considered)
- [5. Assumptions](#5-assumptions)
- [6. Future work](#6-future-work)

---

## 1. Background

Currently, [AGENTS.md (Project Planning)](../../AGENTS.md#L19-L28) specifies conventions for distinguishing direct requests from plans, requiring user approval for planning documents, and enforcing prominent approval callouts. Furthermore, [AGENTS.md (Git Workflow)](../../AGENTS.md#L7-L12) establishes policies for clean commits and remote pushing.

However, neither section defines a structured schema for technical design proposals. As seen in earlier documents like [docs/plans/mvp_plan.md](mvp_plan.md) and [docs/plans/data_representation_cleanup_plan.md](data_representation_cleanup_plan.md), past plans lacked a consistent section hierarchy, omitted formal evaluations of rejected options, had no defined archival strategy when contradicted, and were detached from commit messages.

Standardizing technical design plans according to principles outlined in [Google AIP-1 (Design Documents)](https://google.aip.dev/1), the [GitHub Flavored Markdown Specification](https://github.github.com/gfm/), and [Pro Git Commit Guidelines](https://git-scm.com/book/en/v2/Distributed-Git-Contributing-to-a-Project#_commit_guidelines) will ensure that all architectural decisions at Quadsmith are uniform, rigorously justified, easy to review, and permanently preserved in the repository history.

---

## 2. Goal

Formally codify the standardized plan document structure (including Table of Contents, Proposal blockquoted TL;DR and high-level overview, relative path file links for git, and evidence links), the draft-to-repo lifecycle, the plan superseding/archival process, and the commit message formatting rules directly into [AGENTS.md](../../AGENTS.md).

---

## 3. Proposal

> **TL;DR:** Standardize plan documents and git integration in [AGENTS.md](../../AGENTS.md):
> * **Structure**: 7 standard sections with Table of Contents, blockquoted TL;DR, and decision-specific alternatives.
> * **Lifecycle**: Draft in session artifacts (`&nbsp;` spacing, `file:///` links); commit to `docs/plans/` (pure markdown, relative links).
> * **Commits**: Implement plans in a single commit containing the Goal and Proposal TL;DR in the commit message.
> * **Maintenance**: Keep active plans updated with each commit; move superseded plans into `docs/plans/archive/`.

This proposal establishes a standardized, end-to-end framework for how technical and architectural plans are drafted, structured, reviewed, versioned, maintained, and committed across the Quadsmith project. By introducing a consistent 7-section format and explicit lifecycle rules, plans will clearly communicate the problem, objective, high-level proposal concept, chosen design decisions, and discarded alternatives without ambiguity.

Furthermore, integrating the plan's Goal and Proposal TL;DR directly into git commit messages ensures that the repository history contains clear context for every major change while preserving the full architectural design in `docs/plans/`. Finally, requiring active plans to stay updated while archiving contradicted plans preserves clean documentation while maintaining an immutable historical record.

### Design Decision 1: Plan Document Structure & Section Rules
In [AGENTS.md](../../AGENTS.md) under `## Project Planning`, prescribe the following structure for every plan document:

- **Table of Contents**: A linked list directly under the title containing all Level 1 headings (Sections 1 through 7) present in the document, plus the design decision headings nested under Proposal.
- **Horizontal Separators**: Only add horizontal rules (`---`) between Level 1 headings (never between sub-headings or design decisions).
- **Omission of Empty Optional Sections**: If an optional section (Background, Alternatives considered, Assumptions, Future work, Open questions) contains no entries or content, omit it completely from the document and do not include it in the Table of Contents.
- **Documenting All Discussed Alternatives**: When discussing a plan, document all alternatives explored during the discussion regardless of their quality or viability.
- **Concise Bulleted Summaries**: Use concise bulleted lists of short items whenever helpful to explain complex concepts cleanly and concisely — across TL;DR sections, design decisions, alternative descriptions, and chat responses.
- **Evidence & Clickable Reference Links**: Include as many clickable reference links where appropriate (ideally not bare links, but descriptive anchor text linking as evidence for statements).
  - For file and code references in committed git plans, always use relative paths from the repository root or file location (e.g. `[AGENTS.md](../../AGENTS.md#L19-L28)` or `[docs/plans/mvp_plan.md](mvp_plan.md)`).
  - Prioritize external links by utility:
    1. [Official documentation](https://www.w3.org/standards/) (most useful)
    2. [Wikipedia](https://www.wikipedia.org/) and [StackExchange](https://stackexchange.com/) sites (moderately useful)
    3. YouTube and Reddit community discussions (slightly useful)
1. **Background (optional)**: The reasoning, extra context around the problem and/or current state. If this plan contradicts or supersedes any existing plans, explicitly reference the superseded plans here.
2. **Goal**: A concise explanation of what this proposal tries to achieve (conceptually).
3. **Proposal**: Starts with `> **TL;DR:** <explanation>` formatted inside a blockquote (typically 1 to 2 short sentences, or a concise bulleted list of short items). Immediately following the TL;DR blockquote, include a more in-depth explanation of the high-level concept for the proposal. Then, provide 1 sub-heading per design decision (`### <Design Decision>`), with the proposal explanation directly under it (using diagrams, tables, images, UI screenshots, UI concept images, artifacts, or code snippets with path references as appropriate).
   - Under each design decision sub-heading, an `#### Alternatives considered` section with 1 sub-heading per alternative (`##### <Alternative Name>`). Each alternative must include:
     1. An unbulleted summary of around 2 sentences explaining the alternative and how it differs from the proposal.
     2. An unbulleted `**Pros:**` label, followed by a bulleted list of pros compared to the proposal.
     3. An unbulleted `**Cons:**` label, followed by a bulleted list of cons compared to the proposal.
4. **Alternatives considered (optional)**: A sibling section following the Proposal for evaluating alternative high-level (conceptually broad) approaches. Contains 1 sub-heading per alternative (`### <Alternative Name>`) with an unbulleted ~2 sentence summary, unbulleted `**Pros:**` with bulleted items, and unbulleted `**Cons:**` with bulleted items.
5. **Assumptions (optional)**: A numbered list of unverified or low-confidence assumptions about the problem space, background, current/future behavior of external systems or imported dependencies, etc.
6. **Future work (optional)**: 1 sub-heading per idea (`### <Idea>`) of how the proposal can be extended in the future to provide even more value.
7. **Open questions (optional)**: A numbered list of any unknowns, subjective decisions, or undecided tradeoffs.

Snippet to add to [AGENTS.md](../../AGENTS.md#L20-L28) under `## Project Planning`:
```markdown
- **Plan Document Structure & Lifecycle**:
  - Author and iterate on plans within the agent artifact directory (`<appDataDir>/brain/<conversation-id>/<plan_name>.md`) during review.
  - In artifact files, use `&nbsp;` on blank lines between adjacent paragraphs for preview spacing, and `file:///` scheme links for editor clickability.
  - Upon approval, convert internal file links to repo-relative paths, remove `&nbsp;` spacing, and commit the finalized plan to `docs/plans/<plan_name>.md` using pure standard Markdown.
  - Every plan document must adhere to the standardized structure:
    - **Table of Contents**: Linked list of all Level 1 headings present in the doc, plus nested design decision headings.
    - **Horizontal Separators**: Use `---` only between Level 1 headings.
    - **Omission of Empty Optional Sections**: Omit Background, Alternatives considered, Assumptions, Future work, or Open questions entirely if empty (do not include placeholders like "None", and exclude omitted sections from the Table of Contents).
    - **Document All Discussed Alternatives**: Document all alternatives explored during discussion, regardless of quality or viability.
    - **Concise Bulleted Summaries**: Use concise bulleted short items whenever helpful to explain complex concepts cleanly across plans, alternatives, and chat explanations.
    - **Clickable Reference Links**: Include as many clickable reference links as appropriate using descriptive anchor text as evidence for statements. For committed repository files, use relative links, never absolute paths. External links should prioritize official documentation, Wikipedia, and StackExchange.
    - **1. Background (optional)**: Context, and explicit references to any superseded plans.
    - **2. Goal**
    - **3. Proposal**: Starts with `> **TL;DR:** <explanation>` blockquote (short 1-2 sentence summary or concise bulleted list of short items), followed by a high-level concept explanation paragraph, then 1 sub-heading per design decision (`### <Design Decision>`) with the proposal explanation directly under it (using diagrams, tables, images, UI screenshots, UI concept images, artifacts, or code snippets with path references as appropriate), and nested `#### Alternatives considered` (unbulleted ~2 sentence summary, unbulleted `**Pros:**` and `**Cons:**` labels with bulleted items).
    - **4. Alternatives considered (optional)**: High-level conceptual alternatives.
    - **5. Assumptions (optional)**: Numbered list.
    - **6. Future work (optional)**: 1 sub-heading per idea.
    - **7. Open questions (optional)**: Numbered list.
```

#### Alternatives considered

##### Loose Section Suggestions

Allow flexible heading formats and omit the strict per-decision alternatives structure. This provides maximum flexibility for authors to structure notes however they prefer.

**Pros:**
* Slightly faster to write initial drafts for small changes.

**Cons:**
* Inconsistent formatting across docs.
* Omits critical evaluation of design alternatives.

##### Include Empty Optional Sections as Placeholders

Always include all 7 sections even when empty by adding placeholder text such as "(None)".

**Pros:**
* Enforces an identical, predictable 7-section table of contents across every plan document.

**Cons:**
* Adds noisy, valueless boilerplate to documents.
* Distracts reviewers with empty sections.

### Design Decision 2: Plan Lifecycle (Artifacts to `docs/plans/`)
Define the two-stage lifecycle for all plans:
1. **Discussion & Review Phase (Artifacts)**: Plans are authored and iterated on as artifacts in the agent artifact directory (`<appDataDir>/brain/<conversation-id>/<plan_name>.md`). Insert `&nbsp;` on blank lines between adjacent paragraphs for clean visual spacing in the Antigravity preview. In the artifact document, internal file links use the `file:///` scheme so clicking them opens the local workspace file directly in the editor.
2. **Implementation Phase (Git Repository)**: Once approved, the plan document is created at `docs/plans/<plan_name>.md` and committed along with the implementation code. During this step, strip `&nbsp;` paragraph spacers (restoring pure standard Markdown) and convert internal `file:///` links to repository-relative paths (e.g., `../../AGENTS.md`) to maintain portability across git clones and web viewers.

#### Alternatives considered

##### Direct Authoring in `docs/plans/`

Create and edit the plan file in the git working tree from the start of the conversation. This bypasses the need to move files between directories.

**Pros:**
* Avoids copying files from the artifact directory to the repository.

**Cons:**
* Pollutes `git status` with unapproved, in-progress drafts.
* Violates the project rule in [AGENTS.md (Project Planning)](../../AGENTS.md#L26-L26) that iterations remain confined to preview/artifact directories before approval.

##### Relative File Links in Artifact Previews

Use relative file links like `(AGENTS.md)` directly within the artifact file during planning review.

**Pros:**
* No URL conversion step required when saving the file to `docs/plans/`.

**Cons:**
* Causes "file not found" errors when clicked in the Antigravity preview, because relative links resolve against the artifact directory (`.gemini/.../brain/`) rather than the workspace root.

### Design Decision 3: Git Commit Workflow Integration
In [AGENTS.md](../../AGENTS.md) under `## Git Workflow & Commit Messages`, define the following commit conventions:
1. **Single Commit Implementation**: Plans should be implemented in a single commit.
2. **Commit Message Format**: The commit message must include the standard top-line subject, followed by just the **Goal** section and a **TL;DR of the Proposal** (conforming to [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) and [Git Commit Best Practices](https://stackoverflow.com/questions/2126210/detailed-guidelines-for-git-commit-messages)).
3. **Follow-up Commits**:
   - If a follow-up commit is required because the plan needs to change: update the plan document in `docs/plans/<plan_name>.md` as part of that commit.
   - If a follow-up commit is required because the code does not correctly implement the plan: treat it as a bugfix with a simple commit message (the plan document does not need to be updated).
   - If the follow-up commit is for a plan implemented in the latest commit on the branch: amend the commit via [`git commit --amend`](https://git-scm.com/docs/git-commit#Documentation/git-commit.txt---amend) with the changes (and make any corresponding updates to the plan document and commit message). If already pushed to remote, ask the user before force-pushing via [`git push --force-with-lease`](https://git-scm.com/docs/git-push#Documentation/git-push.txt---force-with-leaseltrefnamegt).

Snippet to add to [AGENTS.md](../../AGENTS.md#L7-L12) under `## Git Workflow & Commit Messages`:
```markdown
- **Plan Implementation & Commits**:
  - Implement plans in a single commit that saves the finalized plan to `docs/plans/<plan_name>.md`.
  - Include the **Goal** section and a **TL;DR of the Proposal** in the commit message body following the subject line.
  - For follow-up commits:
    - If the plan changes: update the plan in `docs/plans/<plan_name>.md` in that commit.
    - If fixing an incorrect implementation: treat it as a standard bugfix commit (no plan document update).
    - If following up immediately on the branch's latest commit: amend the commit and update the plan/message as needed. If already pushed, ask before force-pushing.
```

#### Alternatives considered

##### Embed Full Plan in Commit Message

Append the entire markdown plan (all sections) into the commit message body. This keeps git commit logs completely standalone.

**Pros:**
* Commit message is fully self-contained without needing to view repository files.

**Cons:**
* Produces massive commit logs.
* Causes duplication between git logs and `docs/plans/`.
* Harder to read with `git log --oneline` or terminal viewers.

##### Always Create New Commits for Immediate Follow-ups

Never amend the latest commit when a follow-up is immediately needed for a plan on the branch; always create a new commit.

**Pros:**
* Never requires force-pushing.

**Cons:**
* Pollutes branch history with fragmented "fix" commits instead of a clean, cohesive initial implementation.

### Design Decision 4: Plan Maintenance, Superseding, and Archival
In [AGENTS.md](../../AGENTS.md) under `## Project Planning`, define the rules for ongoing plan maintenance and deprecation:
1. **Keep Active Plans Updated**: All non-archived plans in `docs/plans/` must be kept up-to-date with every new commit, in line with [Software Documentation Lifecycle Best Practices](https://en.wikipedia.org/wiki/Software_documentation).
2. **Superseding Contradicted Plans**:
   - If a new plan contradicts an existing plan, do **not** modify the existing plan documents.
   - Reference the superseded plan(s) in the new plan's `1. Background` section.
   - Consider the superseded plans archived and move them into `docs/plans/archive/` (following standard [software deprecation practices](https://en.wikipedia.org/wiki/Deprecation)).
   - Notify the user of this situation while discussing the plan.
3. **Clarifying Uncertainty**: If unsure whether to archive an existing plan or update it to match new changes, explicitly ask the user.

Snippet to add to [AGENTS.md](../../AGENTS.md#L20-L28) under `## Project Planning`:
```markdown
- **Plan Maintenance & Archival**:
  - Keep all non-archived plans in `docs/plans/` up-to-date with every new commit.
  - If a new plan contradicts an existing plan:
    - Do not modify the existing plan document.
    - Reference the superseded plan(s) in the `Background` section of the new plan.
    - Move superseded plans into `docs/plans/archive/`.
    - Notify the user of this situation during the plan discussion.
  - If unsure whether an existing plan should be updated or archived, ask the user.
```

#### Alternatives considered

##### Edit Superseded Plans In-Place

Directly edit superseded plans in `docs/plans/` to reflect new architecture rather than archiving them.

**Pros:**
* Keeps all documentation in a single flat directory without an archive subfolder.

**Cons:**
* Destroys historical context of why earlier architectural choices were made.
* Makes it difficult to audit past decisions and system evolution.

##### Retain Superseded Plans in `docs/plans/` with Deprecation Headers

Leave superseded plans in `docs/plans/` and prepend a note stating that they have been replaced.

**Pros:**
* Avoids file move operations in git.

**Cons:**
* Clutters the active `docs/plans/` directory with obsolete plans.
* Increases the risk of agents or developers referencing outdated specifications.

### Design Decision 5: Self-Execution for this Change
To apply these rules:
1. Update [AGENTS.md](../../AGENTS.md) with the new sections in `## Git Workflow & Commit Messages` and `## Project Planning`.
2. Ensure `docs/plans/archive/` exists.
3. Save this plan to `docs/plans/plan_structure_and_commit_workflow.md` using pure standard Markdown (stripping `&nbsp;`) and repo-relative file links.
4. Commit files in a single commit whose message includes the Goal and Proposal TL;DR.
5. Push to remote.

#### Alternatives considered

##### Update `AGENTS.md` without Saving this Plan to `docs/plans/`

Only update `AGENTS.md` and omit writing this plan into `docs/plans/`. This minimizes the number of files touched during the task.

**Pros:**
* Touches only one file.

**Cons:**
* Misses the opportunity to establish the first reference document following the new standard.

---

## 4. Alternatives considered

### PR-Driven Plans

Use Pull Request descriptions for plans rather than storing Markdown files in the repository, relying on [GitHub Pull Requests](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests).

**Pros:**
* Keeps the git working tree clean of design docs.

**Cons:**
* Inaccessible when working offline or inspecting local history.
* Hard to version alongside code refactors.

---

## 5. Assumptions

1. The plan document in `docs/plans/<plan_name>.md` should remain formatted in [GitHub Flavored Markdown](https://github.github.com/gfm/).
2. The rules apply to all future planning tasks across the repository.

---

## 6. Future work

### Plan Verification Tooling
- Implement a lightweight check (e.g. `make test` or pre-commit) to verify that any modified or new file in `docs/plans/` follows the required top-level headings (`Goal`, `Proposal`, etc.).
