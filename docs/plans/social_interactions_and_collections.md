# Social Interactions, Q&A, Reviews, and Collections Design

## Table of Contents
- [1. Goal](#1-goal)
- [2. Proposal](#2-proposal)
  - [Polymorphic Relational Foreign Key Strategy](#polymorphic-relational-foreign-key-strategy)
  - [Steam-Style Binary Review Subsystem](#steam-style-binary-review-subsystem)
  - [StackOverflow-Style Technical Q&A Subsystem](#stackoverflow-style-technical-qa-subsystem)
  - [Dual-Authority Answer Endorsement Badges](#dual-authority-answer-endorsement-badges)
  - [Lightweight Clarification Micro-Comments](#lightweight-clarification-micro-comments)
  - [Reputation Karma Economy and Anti-Griefing Safeguards](#reputation-karma-economy-and-anti-griefing-safeguards)
  - [Public Multi-Item Collections and System Likes](#public-multi-item-collections-and-system-likes)
  - [Knowledge Preservation and Soft-Deletion Lifecycle](#knowledge-preservation-and-soft-deletion-lifecycle)
- [3. Assumptions](#3-assumptions)

---

## 1. Goal

Provide a structured, community-driven social architecture for Quadsmith that separates evaluative opinions from technical troubleshooting, incentivizes helpful contributions through a balanced karma economy, preserves communal knowledge, and enables flexible curating across aircraft builds and catalog components.

---

## 2. Proposal

> **TL;DR:**
> - Enable polymorphic social features across aircraft builds and catalog components using strict relational foreign keys with PostgreSQL check constraints.
> - Separate subjective evaluations into Steam-style binary recommendations (thumbs up/down with `% Positive` score tiers) and technical troubleshooting into StackOverflow-style Q&A.
> - Implement dual-authority answer badges: question askers award "Accepted Answer" checkmarks, while build authors confer "Author Verified" endorsement badges.
> - Prevent griefing with a high-stakes karma economy: downvoting costs the voter $-2$ reputation, requires a 10 reputation unlock threshold, and is limited to 3 downvotes per day.
> - Support mixed-item public collections (containing both builds and components) anchored by a 1-click system "Liked" collection.

The social interaction architecture addresses the distinct needs of FPV builders: sharing flight impressions, seeking technical troubleshooting advice (wiring, betaflight tuning, standoff spacing), and bookmarking components for future projects. By architecturally decoupling opinion reviews from Q&A troubleshooting and backing all interactions with relational database constraints, Quadsmith maintains high content quality while eliminating common comment-section toxicity.

### Polymorphic Relational Foreign Key Strategy
Social entities (`Review`, `Question`, `CollectionItem`) associate polymorphically with both assembled `Build` records and individual catalog `Component` entities. Rather than employing unconstrained generic string targets, tables maintain explicit nullable foreign keys (`build_uuid`, `entity_uuid`) enforced by PostgreSQL check constraints:

```sql
CHECK (num_nonnulls(build_uuid, entity_uuid) = 1)
```

This guarantees referential integrity, supports native cascading deletes, and allows foreign key joins without dynamic table lookups.

#### Alternatives considered
Generic polymorphic target columns storing a string `target_type` (e.g., `'BUILD'`, `'COMPONENT'`) and a target UUID.
**Pros:**
- Adds support for new targetable resource types in the future without modifying table DDL.
**Cons:**
- Destroys PostgreSQL relational integrity and precludes foreign key constraints.
- Prevents database-level cascading deletes when builds or catalog components are pruned.

### Steam-Style Binary Review Subsystem
Subjective evaluations are structured as Steam-style recommendations rather than numerical star ratings:
- **Recommendation**: A binary boolean (`is_recommended`) indicating Thumbs Up or Thumbs Down, paired with a Markdown body.
- **Constraints**: Exactly one review per user per target entity; build creators are prohibited from reviewing their own aircraft.
- **Score Aggregation**: Aggregate satisfaction is computed as the percentage of positive recommendations:

$$\text{Score \%} = \left( \frac{\text{Count of Recommended Reviews}}{\text{Total Reviews}} \right) \times 100$$

Displayed alongside qualitative tiers:
- **Overwhelmingly Positive**: $\ge 95\%$ positive ($\ge 50$ reviews)
- **Very Positive**: $80\% - 94\%$ positive ($\ge 10$ reviews)
- **Positive**: $70\% - 79\%$ positive
- **Mixed**: $40\% - 69\%$ positive
- **Mostly Negative**: $20\% - 39\%$ positive
- **Overwhelmingly Negative**: $< 20\%$ positive ($\ge 50$ reviews)
- **Few Reviews**: Rendered as a neutral count until reaching a minimum threshold of 5 reviews.

Review readers can flag reviews as **Helpful** or **Unhelpful** to drive "Most Helpful" sorting without influencing the target entity's positive percentage.

#### Alternatives considered
Traditional 1-to-5 star rating systems.
**Pros:**
- Familiar interface pattern widely used in consumer e-commerce.
**Cons:**
- Suffers from chronic 5-star inflation, rendering intermediate 2, 3, and 4-star ratings ambiguous and difficult to calibrate.
- Blurs the line between general satisfaction and specific technical defects.

### StackOverflow-Style Technical Q&A Subsystem
To prevent technical troubleshooting and fitment guidance from being lost in conversational chatter, technical questions are housed in a dedicated Q&A section:
- **Questions**: Posted against a build or component with a title, markdown body, and net community score.
- **Answers**: Independent submissions per question with separate upvote and downvote voting. Answers are ranked dynamically:
  1. Accepted Answer (pinned first)
  2. Author Endorsed Answer
  3. Net score ($\text{Upvotes} - \text{Downvotes}$)
  4. Chronological recency

#### Alternatives considered
A single, flat chronological comments thread beneath builds and components.
**Pros:**
- Simple to implement and requires minimal database models.
**Cons:**
- Causes valuable wiring solutions, PID tuning tips, and fitment answers to be buried under casual comments.
- Lacks mechanisms to elevate proven solutions or endorse technical accuracy.

### Dual-Authority Answer Endorsement Badges
To balance the perspective of the user who experienced the problem with the technical authority of the builder:
- **Accepted Answer ($\checkmark$ Checkmark)**: Conferred exclusively by the **Question Asker** when a solution resolves their inquiry. Floats to the top of the answer list.
- **Author Endorsed ($\bigstar$ Badge)**: Conferred exclusively by the **Build Author** on their own builds (or catalog administrators on hardware components). Confirms that the proposed advice aligns with the builder's physical testing.

An answer may receive either, both, or neither badge.

### Lightweight Clarification Micro-Comments
Questions and Answers each support flat, chronological micro-comments:
- Restricted to brief clarifications (e.g., verifying firmware version or flight cell count).
- Flat hierarchy without nested replies to prevent recursive discussion trees.
- Micro-comments do not support voting, minimizing overhead and keeping focus on primary answers.

### Reputation Karma Economy and Anti-Griefing Safeguards
User reputation reflects accumulated helpfulness across the community. Points are derived directly from relational vote records and endorsement events:

| Community Action | Author Award | Voter Cost | Safeguards & Caps |
|---|---|---|---|
| **Question Upvoted** | `+5` | `0` | Capped at `+100` per post |
| **Question Downvoted** | `-2` | `-2` | 3 downvotes per rolling 24h |
| **Answer Upvoted** | `+5` | `0` | Capped at `+100` per post |
| **Answer Downvoted** | `-2` | `-2` | 3 downvotes per rolling 24h |
| **Review Voted Helpful** | `+5` | `0` | Capped at `+100` per review |
| **Review Voted Unhelpful** | `-2` | `-2` | 3 downvotes per rolling 24h |
| **Answer Accepted by Asker** | `+30` (Answerer) | `0` (`+10` Asker bonus) | 1 accepted answer per question |
| **Answer Endorsed by Author** | `+20` (Answerer) | `0` | 1 endorsed answer per question |

#### Anti-Griefing Safeguards:
- **Downvote Unlock Threshold**: Users must earn at least **10 reputation** before gaining the privilege to downvote.
- **Voter Skin-in-the-Game**: Every cast downvote costs the voter $-2$ reputation.
- **Daily Rate Ceiling**: Strict limit of 3 downvotes per user in any rolling 24-hour window.
- **Reputation Floor**: Total reputation cannot drop below `0`.

#### Alternatives considered
Free, unconstrained downvoting with zero voter cost and no unlock threshold.
**Pros:**
- Low friction for users to flag incorrect answers or low-effort content.
**Cons:**
- Encourages retaliatory vote brigading and toxic pile-ons from throwaway accounts.

### Public Multi-Item Collections and System Likes
Users can organize and curate builds and components into collections:
- **System "Liked" Collection**: Initialized automatically for every user account upon signup. Clicking the Heart icon on any build or component adds the item to this collection and increments the public item counter.
- **Custom Collections**: Users can create named collections (e.g., *"Sub-250g Cinematic Builds"*, *"6S Freestyle Motors"*) containing mixed item types (both builds and catalog components).
- **Public Visibility**: All collections are public by design and exposed on the user's public profile (`/@username/collections`).

### Knowledge Preservation and Soft-Deletion Lifecycle
To protect communal troubleshooting history:
- **Question Preservation Rule**: A Question with at least one accepted answer or an answer with a net positive score cannot be permanently hard-deleted by the author. If deletion is requested, the author's identity is masked as `[Deleted User]`, while the question and solutions remain accessible.
- **Review Deletions**: An author may delete their review at any time, immediately removing it from the target entity's `% Positive` ratio.
- **Comment Deletions**: Comments with replies display `[Comment deleted]` to maintain conversation context.

---

## 3. Assumptions

1. Social interactions are restricted to authenticated users with verified email addresses.
2. Aircraft builds and catalog components can receive reviews, questions, and collections polymorphically.
3. User reputation is computed deterministically from underlying vote, acceptance, and endorsement records.
