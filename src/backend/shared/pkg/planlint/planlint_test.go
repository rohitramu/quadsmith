package planlint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func findPlansDir() string {
	candidates := []string{
		"docs/plans",
		"../../../../../docs/plans",
		"../../../../docs/plans",
		"../../../docs/plans",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return ""
}

func TestAllRepoPlans(t *testing.T) {
	plansDir := findPlansDir()
	if plansDir == "" {
		t.Fatal("could not locate docs/plans directory")
	}

	files, violations, err := LintDirectory(plansDir)
	if err != nil {
		t.Fatalf("LintDirectory(%q) failed: %v", plansDir, err)
	}
	if len(files) == 0 {
		t.Fatalf("expected at least 1 plan document in %q, found 0", plansDir)
	}

	if len(violations) > 0 {
		var sb strings.Builder
		for _, v := range violations {
			sb.WriteString("\n  • ")
			sb.WriteString(v.String())
		}
		t.Fatalf("found %d plan linting violation(s) across %d file(s):%s", len(violations), len(files), sb.String())
	}

	t.Logf("Successfully verified %d plan document(s) in %s", len(files), plansDir)
}

const validMinimalPlan = `# <a id="sample-architecture-plan" >Sample Architecture Plan</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Goal](#1-goal)
- [2. Proposal](#2-proposal)
  - [Core Engine Design](#core-engine-design)

---

## <a id="1-goal" >1. Goal</a>

Provide a deterministic validation engine for sample architecture plans.

---

## <a id="2-proposal" >2. Proposal</a>

> **TL;DR:**
>
> - Parse markdown documents into structured heading and block spans.
> - Validate every section against [CommonMark](https://commonmark.org/) rules.

The engine inspects every line of the plan document and verifies structural
invariants before allowing commits to complete.

### <a id="core-engine-design" >Core Engine Design</a>

The core engine runs as a fast single-pass validator over the markdown lines.

#### <a id="alternatives-considered-core-engine" >Alternatives considered</a>

##### <a id="manual-human-review" >Manual Human Review</a>

Relying exclusively on manual inspection during code review without automated
checks. Reviewers manually check every heading anchor and section rule.

**Pros:**

- Zero initial implementation effort.

**Cons:**

- Prone to human error and inconsistent enforcement across commits.
`

const validFull7SectionPlan = `# <a id="full-seven-section-plan" >Full Seven Section Plan</a>

## <a id="table-of-contents" >Table of Contents</a>

- [1. Background](#1-background)
- [2. Goal](#2-goal)
- [3. Proposal](#3-proposal)
  - [Primary Subsystem Design](#primary-subsystem-design)
- [4. Alternatives considered](#4-alternatives-considered)
- [5. Assumptions](#5-assumptions)
- [6. Future work](#6-future-work)
- [7. Open questions](#7-open-questions)

---

## <a id="1-background" >1. Background</a>

Prior systems lacked automated structural verification, leading to drift across
[GitHub Flavored Markdown](https://github.github.com/gfm/) documents.

---

## <a id="2-goal" >2. Goal</a>

Enforce all seven standardized sections when present in a technical design plan.

---

## <a id="3-proposal" >3. Proposal</a>

> **TL;DR:** Enforce all seven standard sections with deterministic checks.

The proposal introduces a comprehensive linter covering all optional and
required sections.

### <a id="primary-subsystem-design" >Primary Subsystem Design</a>

The primary subsystem validates headings, tables of contents, and alternatives.

#### <a id="alternatives-primary-subsystem" >Alternatives considered</a>

##### <a id="regex-only-grep-script" >Regex-Only Grep Script</a>

Using a shell script with basic ` + "`grep`" + ` invocations to check top-level headers.
It does not parse section spans or blockquotes.

**Pros:**

- Minimal script size.

**Cons:**

- Cannot validate nested alternatives or Table of Contents ordering.

---

## <a id="4-alternatives-considered" >4. Alternatives considered</a>

### <a id="external-commercial-linter" >External Commercial Linter</a>

Adopting an external SaaS documentation platform instead of repository markdown
files. All plans would live outside git.

**Pros:**

- Built-in WYSIWYG editing templates.

**Cons:**

- Breaks offline access and decouples plans from git commits.

---

## <a id="5-assumptions" >5. Assumptions</a>

1. All active plans are stored in ` + "`docs/plans/`" + `.
2. Archived plans live in ` + "`docs/plans/archive/`" + `.

---

## <a id="6-future-work" >6. Future work</a>

### <a id="ide-diagnostics-extension" >IDE Diagnostics Extension</a>

Surface plan linter violations inline within the code editor in real time.

---

## <a id="7-open-questions" >7. Open questions</a>

1. Should the linter support auto-fixing missing Table of Contents entries?
2. Should warning severity be added for external link domain prioritization?
`

func TestValidPlans(t *testing.T) {
	if v := LintContent(validMinimalPlan, Options{SkipFileExistenceCheck: true}); len(v) != 0 {
		t.Errorf("expected 0 violations for validMinimalPlan, got %v", v)
	}
	if v := LintContent(validFull7SectionPlan, Options{SkipFileExistenceCheck: true}); len(v) != 0 {
		t.Errorf("expected 0 violations for validFull7SectionPlan, got %v", v)
	}
}

func TestArtifactMode(t *testing.T) {
	artifactPlan := strings.ReplaceAll(validMinimalPlan,
		"> - Validate every section against [CommonMark](https://commonmark.org/) rules.",
		"> - Validate every section against\n>   [AGENTS.md](file:///home/roramu/projects/quadsmith/AGENTS.md) rules.")
	artifactPlan = strings.Replace(artifactPlan,
		"The engine inspects every line of the plan document and verifies structural\ninvariants before allowing commits to complete.\n",
		"The engine inspects every line of the plan document and verifies structural\ninvariants before allowing commits to complete.\n\n&nbsp;\n\nSecond paragraph in artifact preview.\n",
		1)

	// In artifact mode, &nbsp; and file:/// links are valid.
	if v := LintContent(artifactPlan, Options{IsArtifact: true, SkipFileExistenceCheck: true}); len(v) != 0 {
		t.Errorf("expected 0 violations in artifact mode, got %v", v)
	}

	// In committed mode, &nbsp; and file:/// links must trigger RuleLifecycleFormat.
	committedViolations := LintContent(artifactPlan, Options{IsArtifact: false, SkipFileExistenceCheck: true})
	if !hasRuleViolation(committedViolations, RuleLifecycleFormat) {
		t.Errorf("expected %s violation when linting artifact plan in committed mode, got %v", RuleLifecycleFormat, committedViolations)
	}
}

func TestRulesCatchViolations(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(string) string
		wantRule RuleID
	}{
		{
			name: "missing H1 title on line 1",
			mutate: func(s string) string {
				return strings.Replace(s, "# <a id=\"sample-architecture-plan\" >Sample Architecture Plan</a>", "## <a id=\"sample-architecture-plan\" >Sample Architecture Plan</a>", 1)
			},
			wantRule: RuleDocumentTitle,
		},
		{
			name: "missing custom anchor on heading",
			mutate: func(s string) string {
				return strings.Replace(s, "### <a id=\"core-engine-design\" >Core Engine Design</a>", "### Core Engine Design", 1)
			},
			wantRule: RuleHeadingAnchor,
		},
		{
			name: "duplicate heading anchor id",
			mutate: func(s string) string {
				return strings.Replace(s, "##### <a id=\"manual-human-review\" >Manual Human Review</a>", "##### <a id=\"core-engine-design\" >Manual Human Review</a>", 1)
			},
			wantRule: RuleHeadingAnchor,
		},
		{
			name: "table of contents mismatch",
			mutate: func(s string) string {
				return strings.Replace(s, "  - [Core Engine Design](#core-engine-design)", "  - [Wrong Title](#core-engine-design)", 1)
			},
			wantRule: RuleTableOfContents,
		},
		{
			name: "horizontal separator between sub-headings",
			mutate: func(s string) string {
				return strings.Replace(s, "#### <a id=\"alternatives-considered-core-engine\" >Alternatives considered</a>", "---\n\n#### <a id=\"alternatives-considered-core-engine\" >Alternatives considered</a>", 1)
			},
			wantRule: RuleHorizontalSeparator,
		},
		{
			name: "missing horizontal separator before Level 1 section",
			mutate: func(s string) string {
				return strings.Replace(s, "---\n\n## <a id=\"2-proposal\" >2. Proposal</a>", "## <a id=\"2-proposal\" >2. Proposal</a>", 1)
			},
			wantRule: RuleHorizontalSeparator,
		},
		{
			name: "missing blank line before heading",
			mutate: func(s string) string {
				return strings.Replace(s, "\n\n### <a id=\"core-engine-design\" >Core Engine Design</a>", "\n### <a id=\"core-engine-design\" >Core Engine Design</a>", 1)
			},
			wantRule: RuleBlankLines,
		},
		{
			name: "missing blank line before bulleted list",
			mutate: func(s string) string {
				return strings.Replace(s, "**Pros:**\n\n- Zero initial implementation effort.", "**Pros:**\n- Zero initial implementation effort.", 1)
			},
			wantRule: RuleBlankLines,
		},
		{
			name: "line over 80 characters that can wrap",
			mutate: func(s string) string {
				return strings.Replace(s,
					"Provide a deterministic validation engine for sample architecture plans.",
					"Provide a deterministic validation engine for sample architecture plans that exceeds eighty characters on a single line.",
					1)
			},
			wantRule: RuleLineLength,
		},
		{
			name: "wrong top-level section numbering",
			mutate: func(s string) string {
				s = strings.Replace(s, "- [2. Proposal](#2-proposal)", "- [3. Proposal](#3-proposal)", 1)
				return strings.Replace(s, "## <a id=\"2-proposal\" >2. Proposal</a>", "## <a id=\"3-proposal\" >3. Proposal</a>", 1)
			},
			wantRule: RuleTopLevelSections,
		},
		{
			name: "empty optional section with None placeholder",
			mutate: func(s string) string {
				s = strings.Replace(s, "  - [Core Engine Design](#core-engine-design)", "  - [Core Engine Design](#core-engine-design)\n- [3. Assumptions](#3-assumptions)", 1)
				return s + "\n---\n\n## <a id=\"3-assumptions\" >3. Assumptions</a>\n\nNone\n"
			},
			wantRule: RuleEmptySection,
		},
		{
			name: "TL;DR statement exceeding 20 words",
			mutate: func(s string) string {
				return strings.Replace(s,
					"> - Parse markdown documents into structured heading and block spans.",
					"> - Parse markdown documents into structured heading and block spans with more than twenty words in this single bulleted statement right here today.",
					1)
			},
			wantRule: RuleProposalTLDR,
		},
		{
			name: "TL;DR with more than 5 statements",
			mutate: func(s string) string {
				return strings.Replace(s,
					"> - Parse markdown documents into structured heading and block spans.",
					"> - Statement one.\n> - Statement two.\n> - Statement three.\n> - Statement four.\n> - Statement five.\n> - Statement six.",
					1)
			},
			wantRule: RuleProposalTLDR,
		},
		{
			name: "missing high-level overview paragraph in Proposal",
			mutate: func(s string) string {
				return strings.Replace(s,
					"The engine inspects every line of the plan document and verifies structural\ninvariants before allowing commits to complete.\n\n",
					"",
					1)
			},
			wantRule: RuleProposalOverview,
		},
		{
			name: "missing H5 sub-heading under decision Alternatives considered",
			mutate: func(s string) string {
				return strings.Replace(s, "##### <a id=\"manual-human-review\" >Manual Human Review</a>\n\n", "", 1)
			},
			wantRule: RuleDecisionAlternatives,
		},
		{
			name: "alternative missing Cons section",
			mutate: func(s string) string {
				return strings.Replace(s, "\n**Cons:**\n\n- Prone to human error and inconsistent enforcement across commits.\n", "\n", 1)
			},
			wantRule: RuleDecisionAlternatives,
		},
		{
			name: "Assumptions section with bulleted list instead of numbered list",
			mutate: func(_ string) string {
				return strings.Replace(validFull7SectionPlan,
					"1. All active plans are stored in `docs/plans/`.\n2. Archived plans live in `docs/plans/archive/`.",
					"- All active plans are stored in `docs/plans/`.",
					1)
			},
			wantRule: RuleAssumptionsFormat,
		},
		{
			name: "Future work section without H3 idea sub-heading",
			mutate: func(_ string) string {
				return strings.Replace(validFull7SectionPlan,
					"### <a id=\"ide-diagnostics-extension\" >IDE Diagnostics Extension</a>\n\n",
					"",
					1)
			},
			wantRule: RuleFutureWorkFormat,
		},
		{
			name: "Open questions section without numbered list",
			mutate: func(_ string) string {
				return strings.Replace(validFull7SectionPlan,
					"1. Should the linter support auto-fixing missing Table of Contents entries?\n2. Should warning severity be added for external link domain prioritization?",
					"Should the linter support auto-fixing?",
					1)
			},
			wantRule: RuleOpenQuestionsFormat,
		},
		{
			name: "bare URL in prose",
			mutate: func(s string) string {
				return strings.Replace(s, "[CommonMark](https://commonmark.org/)", "https://commonmark.org/", 1)
			},
			wantRule: RuleReferenceLinks,
		},
		{
			name: "non-descriptive link text",
			mutate: func(s string) string {
				return strings.Replace(s, "[CommonMark](https://commonmark.org/)", "[here](https://commonmark.org/)", 1)
			},
			wantRule: RuleReferenceLinks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.mutate(validMinimalPlan)
			violations := LintContent(doc, Options{SkipFileExistenceCheck: true})
			if !hasRuleViolation(violations, tt.wantRule) {
				t.Errorf("expected violation of rule %s, got violations: %v", tt.wantRule, violations)
			}
		})
	}
}

func TestRelativeLinkExistenceCheck(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "test_plan.md")

	docWithBrokenLink := strings.ReplaceAll(validMinimalPlan,
		"[CommonMark](https://commonmark.org/)",
		"[Missing File](non_existent_file.go)")

	violations := LintContent(docWithBrokenLink, Options{
		FilePath:               planPath,
		SkipFileExistenceCheck: false,
	})
	if !hasRuleViolation(violations, RuleReferenceLinks) {
		t.Errorf("expected %s violation for broken relative file link, got %v", RuleReferenceLinks, violations)
	}

	// Create the referenced file and verify it passes
	existingFile := filepath.Join(tmpDir, "existing_file.go")
	if err := os.WriteFile(existingFile, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	docWithValidLink := strings.ReplaceAll(validMinimalPlan,
		"[CommonMark](https://commonmark.org/)",
		"[Existing File](existing_file.go#L1)")

	violations = LintContent(docWithValidLink, Options{
		FilePath:               planPath,
		SkipFileExistenceCheck: false,
	})
	if len(violations) != 0 {
		t.Errorf("expected 0 violations when relative link exists, got %v", violations)
	}
}

func hasRuleViolation(violations []Violation, rule RuleID) bool {
	for _, v := range violations {
		if v.Rule == rule {
			return true
		}
	}
	return false
}
