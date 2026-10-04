# Social Interactions, Q&A, Reviews, and Collections Design

## Summary of Decisions

| Decision | Choice | Rationale |
|---|---|---|
| **Target Entities** | `Build` and `Component` | Unified polymorphic social interactions across full aircraft builds and individual catalog components |
| **Polymorphic Database Strategy** | Explicit Foreign Keys (`buildId?`, `componentId?`) | Enforces PostgreSQL foreign key constraints; uses `CHECK (num_nonnulls(buildId, componentId) = 1)` |
| **Review System** | Steam-style Recommendations | Binary Thumbs Up / Down recommendation + text writeup; 1 review per user per entity; author cannot review their own build; aggregate rating is displayed as `% Positive` + qualitative tier |
| **Review Voting** | Helpful / Unhelpful | Binary feedback on review quality; drives "Most Helpful" sorting; comments under reviews are chronological discussion threads without individual comment voting |
| **Discussion & Technical Help** | Dedicated Q&A Section (StackOverflow-style) | Separated from Reviews to keep technical troubleshooting, wiring questions, and fitment guidance organized and searchable |
| **Q&A Structure** | Questions + Independent Answers | Questions and Answers each have Up/Down voting; sorted by net score (`upvotes - downvotes`); accepted/endorsed answers float to the top |
| **Answer Endorsements** | Dual Authority with Distinct Badges | **Question Asker** awards the "Accepted Answer" checkmark; **Build Author** awards the "Author Verified / Endorsed" badge |
| **Micro-Comments** | Lightweight Clarifications | Flat chronological micro-comments attached directly under Questions or Answers for quick clarifications |
| **Reputation System** | StackOverflow-style Karma | Users earn public profile reputation for high-quality community contributions |
| **Reputation Values** | Upvotes: `+5`, Downvotes: `-2`, Max `+100` per post | Asker accepting answer: `+10`; Answer accepted: `+30`; Build author endorsement: `+20` |
| **Anti-Griefing & Downvote Cost** | Downvoting costs `-2` rep to voter; max 3 downvotes/day; 10 rep unlock threshold | Discourages brigading and throwaway accounts; downvoting has real stakes |
| **Content Lifecycle & Deletion** | Knowledge Preservation (StackOverflow/Reddit hybrid) | Questions with accepted or upvoted answers cannot be destroyed; author can disassociate/mask as `[Deleted User]`; deleted comments with replies show `[Comment deleted]`; reviews can be deleted (score recalculates) |
| **Favorites / Collections** | Instagram & Printables style | Default system collection: "Liked" (1-click Heart icon, public by default with privacy toggle); custom collections default to Private; mixed item support (Builds + Components) |

---

## 1. Reviews Subsystem (Steam-Style)

### 1.1 Overview
Top-level evaluation of a Build or Component is modeled as a **Review**. This separates opinions and ratings from troubleshooting and Q&A.

- **Recommendation:** Binary boolean `isRecommended` — `true` (Thumbs Up / Recommended) or `false` (Thumbs Down / Not Recommended).
- **Review Content:** Markdown writeup describing flight performance, durability, ease of build, or component reliability.
- **Constraints:**
  - Exactly **1 review per user per target entity**.
  - A user can update/edit their review at any time. When edited, an `(Edited)` badge and timestamp are displayed.
  - **Self-Review Prevention:** A `Build` author cannot submit a review for their own build. (On `Component` records, all verified users may review).
  - Review author cannot vote on their own review.

### 1.2 Aggregated Score & Tiers
Aggregate ratings are calculated dynamically or materialized via triggers:
$$\text{Score \%} = \left( \frac{\text{Count of Recommended Reviews}}{\text{Total Reviews}} \right) \times 100$$

Displayed with qualitative tiers (matching Steam thresholds):
- **Overwhelmingly Positive:** $\ge 95\%$ positive (minimum 50 reviews)
- **Very Positive:** $80\% - 94\%$ positive (minimum 10 reviews)
- **Positive:** $70\% - 79\%$ positive
- **Mixed:** $40\% - 69\%$ positive
- **Mostly Negative:** $20\% - 39\%$ positive
- **Overwhelmingly Negative:** $< 20\%$ positive (minimum 50 reviews)
- **Few Reviews:** Displayed as "X User Reviews" until a minimum threshold (e.g. 5 reviews) is met.

### 1.3 Review Feedback & Discussion
- **Review Votes:** Readers can vote a review as **Helpful** or **Unhelpful**.
  - Review votes do not impact the entity's % positive score; they determine review sorting ("Most Helpful" vs "Most Recent").
  - Users can change or withdraw their vote.
- **Review Comments:** Each review has a chronological discussion thread for readers to reply directly to the reviewer (e.g., discussing their findings or asking for more flight details).
  - To match Steam review threads, comments under reviews are purely conversational without nested replies or individual comment voting.

---

## 2. Technical Q&A Subsystem (StackOverflow-Style)

### 2.1 Questions & Answers
For technical inquiries ("What length standoffs fit the DJI O3 air unit?", "What Betaflight master multiplier works for these motors?"), a structured Q&A tab replaces unstructured comment threads.

- **Question:** Posted against a `Build` or `Component`.
  - Has a title, body (markdown), and optional tag references.
  - Can be upvoted or downvoted by the community.
- **Answers:** Any user can submit a solution/answer to an open question.
  - Multiple answers can be submitted per question.
  - Answers have independent Up/Down voting.
  - Sorted by:
    1. Accepted Answer (pinned first)
    2. Author Endorsed Answer
    3. Net score ($\text{Upvotes} - \text{Downvotes}$)
    4. Creation date (newest first)

### 2.2 Dual-Authority Endorsements
To recognize accurate answers while acknowledging the build creator's unique authority:
1. **Accepted Answer ($\checkmark$ Checkmark):**
   - Conferred by the **Question Asker**.
   - Indicates that the answer resolved the asker's specific problem.
   - Pinned to the top of the answer feed.
2. **Author Verified / Endorsement ($\bigstar$ Author Badge):**
   - Conferred exclusively by the **Build Author** on their own builds.
   - Indicates: *"The creator of this build has verified that this advice/configuration is correct."*
   - On catalog `Component` entries, this privilege belongs to site moderators/admins.
- An answer can possess either, both, or neither of these badges.

### 2.3 Clarification Micro-Comments
Questions and Answers each support lightweight clarification comments underneath them:
- Designed for minor questions ("Did you test this on 4S or 6S?") rather than full answers.
- Single-level flat list with `@mention` capability.
- No voting on micro-comments to keep overhead minimal.

---

## 3. Reputation & Voting Economy

### 3.1 Point Values & Caps
To incentivize helpful answers and avoid toxic downvote spam:

| Event | Points Awarded to Author | Points Cost to Voter | Cap / Restrictions |
|---|---|---|---|
| **Question Upvoted** | `+5` | `0` | Max `+100` rep per post |
| **Question Downvoted** | `-2` | `-2` | 3 downvotes / rolling 24h |
| **Answer Upvoted** | `+5` | `0` | Max `+100` rep per post |
| **Answer Downvoted** | `-2` | `-2` | 3 downvotes / rolling 24h |
| **Review Voted Helpful** | `+5` | `0` | Max `+100` rep per review |
| **Review Voted Unhelpful** | `-2` | `-2` | 3 downvotes / rolling 24h |
| **Answer Accepted** (by asker) | `+30` (to answerer) | `0` (`+10` bonus to asker) | 1 accepted answer per question |
| **Answer Endorsed** (by build author) | `+20` (to answerer) | `0` | 1 endorsed answer per question |

- **Derivation from Source Tables:** Point events are derived directly from the primary relational vote tables (`QuestionVote`, `AnswerVote`, `ReviewVote`) and answer acceptances/endorsements rather than a separate untyped ledger table, guaranteeing foreign key integrity, cascade deletions, and zero data drift.

### 3.2 Privileges & Anti-Griefing Safeguards
- **Starting Reputation:** New users start at `0` reputation.
- **Minimum Floor:** A user's total reputation cannot drop below `0`.
- **Downvote Unlock Threshold:** A user must achieve at least **`10` reputation** before they unlock the ability to downvote. Upvoting, asking questions, reviewing, and answering are open to all email-verified users immediately.
- **Downvote Rate Limit:** Maximum of **3 downvotes per rolling 24-hour window** per user.
- **Vote Retraction:** If a user cancels an upvote or downvote, all associated reputation adjustments are cleanly reversed.

---

## 4. Favorites & Collections Subsystem (Instagram / Printables-Style)

### 4.1 System "Liked" Collection
- Every user account receives a system-managed **"Liked"** collection upon signup.
- Clicking the **Heart / Like** button on any Build or Component:
  - Adds the item to the user's "Liked" collection.
  - Increments the public Like counter on that entity.
  - Clicking again removes it from "Liked" and decrements the counter.
- **Default Privacy:** The "Liked" collection is **public by default (`isPublic = true`)** (displaying on the user's public profile), but can be toggled to private in profile settings.

### 4.2 Custom Collections
Users can create unlimited custom collections (e.g. *"Sub-250g Ideas"*, *"7-inch Long Range Inspo"*, *"Compatible Motors for Apex 5"*).
- **Default Privacy:** Custom collections default to **private (`isPublic = false`)**.
- **Item Flexibility:** Collections support **mixed items** — a single collection can contain both `Build` and `Component` entries.
- **Many-to-Many:** An item can be added to multiple collections simultaneously.
- **Metadata:** Name, description, `isPublic` (boolean, defaults to `false`), and optional cover image (or auto-preview from member items).
- **Public Profile:** Public collections appear on the user's profile at `/@username/collections`. For this release, public collections are view-only (no collection following or collaborative editing).

---

## 5. Content Moderation, Preservation & Soft-Deletion

- **Knowledge Preservation Rule:** If a Question has at least one accepted answer or an answer with net upvotes ($\ge 1$), the Question cannot be hard-deleted by the asker.
  - If the author requests deletion, the Question is marked with `deletedAt` and author display is masked as `[Deleted User]`. The answers and thread remain intact for the community.
- **Review Deletion:** An author can delete their Review at any time. When deleted, the review is soft-deleted or removed from calculation, and the entity's `% Positive` score immediately recalculates.
- **Comments with Children:** Micro-comments or review comments that have sub-replies display `[Comment deleted]` when removed by the author, preserving conversation context.
- **Account Deletion / Ban:** When a user account is deleted or banned (`bannedAt` / `deletedAt` in `User` table), their contributions remain preserved with the author shown as `[Deleted User]`.

---

## 6. Database Schema Specifications

The following conceptual data models (using pseudo-schema syntax) reflect these architectural decisions:

```prisma
// ==========================================
// 1. REVIEWS & REVIEW COMMENTS (STEAM-STYLE)
// ==========================================

model Review {
  id             Int             @id @default(autoincrement())
  userId         Int
  user           User            @relation(fields: [userId], references: [id], onDelete: Cascade)
  
  // Polymorphic Target (Enforced: exactly one is set)
  buildId        Int?
  build          Build?          @relation(fields: [buildId], references: [id], onDelete: Cascade)
  componentId    Int?
  component      Component?      @relation(fields: [componentId], references: [id], onDelete: Cascade)

  isRecommended  Boolean         // true = Thumbs Up / Recommended, false = Thumbs Down
  body           String          // Markdown writeup
  isEdited       Boolean         @default(false)
  deletedAt      DateTime?
  createdAt      DateTime        @default(now())
  updatedAt      DateTime        @updatedAt

  votes          ReviewVote[]
  comments       ReviewComment[]

  @@unique([userId, buildId])
  @@unique([userId, componentId])
  @@index([buildId, isRecommended])
  @@index([componentId, isRecommended])
}

model ReviewVote {
  id        Int      @id @default(autoincrement())
  reviewId  Int
  review    Review   @relation(fields: [reviewId], references: [id], onDelete: Cascade)
  userId    Int
  user      User     @relation(fields: [userId], references: [id], onDelete: Cascade)
  isHelpful Boolean  // true = Helpful, false = Unhelpful
  createdAt DateTime @default(now())

  @@unique([userId, reviewId])
}

model ReviewComment {
  id        Int       @id @default(autoincrement())
  reviewId  Int
  review    Review    @relation(fields: [reviewId], references: [id], onDelete: Cascade)
  userId    Int
  user      User      @relation(fields: [userId], references: [id], onDelete: Cascade)
  body      String
  deletedAt DateTime?
  createdAt DateTime  @default(now())
  updatedAt DateTime  @updatedAt

  @@index([reviewId, createdAt])
}

// ==========================================
// 2. TECHNICAL Q&A (STACKOVERFLOW-STYLE)
// ==========================================

model Question {
  id          Int            @id @default(autoincrement())
  userId      Int
  user        User           @relation(fields: [userId], references: [id], onDelete: Cascade)
  
  // Polymorphic Target
  buildId     Int?
  build       Build?         @relation(fields: [buildId], references: [id], onDelete: Cascade)
  componentId Int?
  component   Component?     @relation(fields: [componentId], references: [id], onDelete: Cascade)

  title       String
  body        String         // Markdown
  netScore    Int            @default(0) // Denormalized for fast sorting
  isEdited    Boolean        @default(false)
  deletedAt   DateTime?
  createdAt   DateTime       @default(now())
  updatedAt   DateTime       @updatedAt

  answers     Answer[]
  votes       QuestionVote[]
  comments    QuestionComment[]

  @@index([buildId, netScore])
  @@index([componentId, netScore])
}

model Answer {
  id                Int             @id @default(autoincrement())
  questionId        Int
  question          Question        @relation(fields: [questionId], references: [id], onDelete: Cascade)
  userId            Int
  user              User            @relation(fields: [userId], references: [id], onDelete: Cascade)

  body              String          // Markdown
  netScore          Int             @default(0) // Denormalized for fast sorting
  isAcceptedByAsker Boolean         @default(false) // Checkmark from Question Asker
  isAuthorEndorsed  Boolean         @default(false) // Star badge from Build Author
  isEdited          Boolean         @default(false)
  deletedAt         DateTime?
  createdAt         DateTime        @default(now())
  updatedAt         DateTime        @updatedAt

  votes             AnswerVote[]
  comments          AnswerComment[]

  @@index([questionId, netScore])
}

model QuestionVote {
  id         Int      @id @default(autoincrement())
  questionId Int
  question   Question @relation(fields: [questionId], references: [id], onDelete: Cascade)
  userId     Int
  user       User     @relation(fields: [userId], references: [id], onDelete: Cascade)
  isUpvote   Boolean  // true = Upvote (+1), false = Downvote (-1)
  createdAt  DateTime @default(now())

  @@unique([userId, questionId])
}

model AnswerVote {
  id        Int      @id @default(autoincrement())
  answerId  Int
  answer    Answer   @relation(fields: [answerId], references: [id], onDelete: Cascade)
  userId    Int
  user      User     @relation(fields: [userId], references: [id], onDelete: Cascade)
  isUpvote  Boolean  // true = Upvote (+1), false = Downvote (-1)
  createdAt DateTime @default(now())

  @@unique([userId, answerId])
}

model QuestionComment {
  id         Int       @id @default(autoincrement())
  questionId Int
  question   Question  @relation(fields: [questionId], references: [id], onDelete: Cascade)
  userId     Int
  user       User      @relation(fields: [userId], references: [id], onDelete: Cascade)
  body       String
  deletedAt  DateTime?
  createdAt  DateTime  @default(now())
  updatedAt  DateTime  @updatedAt

  @@index([questionId, createdAt])
}

model AnswerComment {
  id        Int       @id @default(autoincrement())
  answerId  Int
  answer    Answer    @relation(fields: [answerId], references: [id], onDelete: Cascade)
  userId    Int
  user      User      @relation(fields: [userId], references: [id], onDelete: Cascade)
  body      String
  deletedAt DateTime?
  createdAt DateTime  @default(now())
  updatedAt DateTime  @updatedAt

  @@index([answerId, createdAt])
}

// ==========================================
// 3. REPUTATION & USER KARMA
// ==========================================

model UserReputation {
  userId        Int      @id
  user          User     @relation(fields: [userId], references: [id], onDelete: Cascade)
  score         Int      @default(0) // Total earned reputation (minimum 0)
  updatedAt     DateTime @updatedAt
}

// ==========================================
// 4. FAVORITES & COLLECTIONS
// ==========================================

model Collection {
  id            Int             @id @default(autoincrement())
  userId        Int
  user          User            @relation(fields: [userId], references: [id], onDelete: Cascade)
  name          String
  description   String?
  isPublic      Boolean         @default(false)
  isDefaultLiked Boolean        @default(false) // System collection for 1-click likes
  createdAt     DateTime        @default(now())
  updatedAt     DateTime        @updatedAt

  items         CollectionItem[]

  @@unique([userId, isDefaultLiked])
  @@index([userId, isPublic])
}

model CollectionItem {
  id           Int        @id @default(autoincrement())
  collectionId Int
  collection   Collection @relation(fields: [collectionId], references: [id], onDelete: Cascade)

  // Polymorphic Target (Build or Component)
  buildId      Int?
  build        Build?     @relation(fields: [buildId], references: [id], onDelete: Cascade)
  componentId  Int?
  component    Component? @relation(fields: [componentId], references: [id], onDelete: Cascade)

  createdAt    DateTime   @default(now())

  @@unique([collectionId, buildId])
  @@unique([collectionId, componentId])
}
```
