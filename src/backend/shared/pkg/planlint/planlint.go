package planlint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// RuleID identifies a specific plan document linting rule.
type RuleID string

const (
	RuleDocumentTitle        RuleID = "DOCUMENT_TITLE"
	RuleHeadingAnchor        RuleID = "HEADING_ANCHOR"
	RuleTableOfContents      RuleID = "TABLE_OF_CONTENTS"
	RuleHorizontalSeparator  RuleID = "HORIZONTAL_SEPARATOR"
	RuleBlankLines           RuleID = "BLANK_LINES"
	RuleLineLength           RuleID = "LINE_LENGTH"
	RuleTopLevelSections     RuleID = "TOP_LEVEL_SECTIONS"
	RuleEmptySection         RuleID = "EMPTY_SECTION"
	RuleProposalTLDR         RuleID = "PROPOSAL_TLDR"
	RuleProposalOverview     RuleID = "PROPOSAL_OVERVIEW"
	RuleProposalDecisions    RuleID = "PROPOSAL_DECISIONS"
	RuleDecisionAlternatives RuleID = "DECISION_ALTERNATIVES"
	RuleTopLevelAlternatives RuleID = "TOP_LEVEL_ALTERNATIVES"
	RuleAssumptionsFormat    RuleID = "ASSUMPTIONS_FORMAT"
	RuleFutureWorkFormat     RuleID = "FUTURE_WORK_FORMAT"
	RuleOpenQuestionsFormat  RuleID = "OPEN_QUESTIONS_FORMAT"
	RuleLifecycleFormat      RuleID = "LIFECYCLE_FORMAT"
	RuleReferenceLinks       RuleID = "REFERENCE_LINKS"
)

// Violation represents a single linting error in a plan document.
type Violation struct {
	File    string
	Line    int
	Rule    RuleID
	Message string
}

func (v Violation) String() string {
	if v.File != "" {
		return fmt.Sprintf("%s:%d [%s] %s", v.File, v.Line, v.Rule, v.Message)
	}
	return fmt.Sprintf("line %d [%s] %s", v.Line, v.Rule, v.Message)
}

// Options configures how a plan document is validated.
type Options struct {
	// FilePath is the path to the file being linted (used for resolving relative links and diagnostics).
	FilePath string
	// IsArtifact indicates the plan is a draft artifact in the agent brain directory
	// (which permits &nbsp; paragraph spacing and requires file:/// internal links)
	// rather than a committed repository plan in docs/plans/.
	IsArtifact bool
	// SkipFileExistenceCheck disables checking that relative file links exist on disk (useful in unit tests).
	SkipFileExistenceCheck bool
}

type parsedHeading struct {
	line      int
	level     int
	anchorID  string
	title     string
	rawLine   string
	validForm bool
}

type canonicalSection struct {
	name     string
	slug     string
	required bool
}

var canonicalSections = []canonicalSection{
	{name: "Background", slug: "background", required: false},
	{name: "Goal", slug: "goal", required: true},
	{name: "Proposal", slug: "proposal", required: true},
	{name: "Alternatives considered", slug: "alternatives-considered", required: false},
	{name: "Assumptions", slug: "assumptions", required: false},
	{name: "Future work", slug: "future-work", required: false},
	{name: "Open questions", slug: "open-questions", required: false},
}

var (
	rawHeadingRegex      = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	anchoredHeadingRegex = regexp.MustCompile(`^(#{1,6}) <a id="([a-z0-9]+(?:[-_][a-z0-9]+)*)" >([^<]+)</a>$`)
	numberedSectionRegex = regexp.MustCompile(`^([0-9]+)\. (.+)$`)
	tocTopItemRegex      = regexp.MustCompile(`^- \[([^\]]+)\]\(#([a-z0-9_-]+)\)$`)
	tocSubItemRegex      = regexp.MustCompile(`^  - \[([^\]]+)\]\(#([a-z0-9_-]+)\)$`)
	unorderedListRegex   = regexp.MustCompile(`^(\s*)[-*+] +(.*)$`)
	orderedListRegex     = regexp.MustCompile(`^(\s*)([0-9]+)\. +(.*)$`)
	markdownLinkRegex    = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)]+)\)`)
	inlineCodeRegex      = regexp.MustCompile("`[^`]*`")
	inlineMathRegex      = regexp.MustCompile(`\$[^$]+\$`)
	bareURLRegex         = regexp.MustCompile(`https?://[^\s)>]+`)
	placeholderTextRegex = regexp.MustCompile(`(?i)^(\(none\)|none\.?|n/a|tbd|nil|nothing\.?|no [a-z ]+\.?)$`)
)

// LintFile reads and validates a plan markdown file on disk.
func LintFile(path string, opts Options) ([]Violation, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	if opts.FilePath == "" {
		opts.FilePath = path
	}
	return LintContent(string(raw), opts), nil
}

// LintDirectory validates all non-archived .md plan files in the given directory (e.g. "docs/plans").
func LintDirectory(dir string) ([]string, []Violation, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("read directory %s: %w", dir, err)
	}

	var checkedFiles []string
	var allViolations []Violation

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		checkedFiles = append(checkedFiles, fullPath)
		v, err := LintFile(fullPath, Options{FilePath: fullPath})
		if err != nil {
			return checkedFiles, allViolations, err
		}
		allViolations = append(allViolations, v...)
	}

	return checkedFiles, allViolations, nil
}

// LintContent validates the markdown contents of a plan document against repository rules.
func LintContent(content string, opts Options) []Violation {
	var violations []Violation
	addViolation := func(line int, rule RuleID, format string, args ...any) {
		violations = append(violations, Violation{
			File:    opts.FilePath,
			Line:    line,
			Rule:    rule,
			Message: fmt.Sprintf(format, args...),
		})
	}

	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		addViolation(1, RuleDocumentTitle, "plan document is empty")
		return violations
	}

	// Track which lines are inside fenced code blocks or multi-line math blocks.
	inCodeBlock := make([]bool, len(lines))
	inMathBlock := make([]bool, len(lines))
	codeFenceOpen := false
	codeFenceStartLine := 0
	mathBlockOpen := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock[i] = true
			if !codeFenceOpen {
				codeFenceOpen = true
				codeFenceStartLine = i + 1
				// Blank line before opening code block
				if i > 0 && !isBlankOrEmptyBlockquote(lines[i-1], opts.IsArtifact) {
					addViolation(i+1, RuleBlankLines, "fenced code block must have a blank line before it")
				}
			} else {
				codeFenceOpen = false
				// Blank line after closing code block (if not last line)
				if i+1 < len(lines) && !isBlankOrEmptyBlockquote(lines[i+1], opts.IsArtifact) {
					addViolation(i+1, RuleBlankLines, "fenced code block must have a blank line after it")
				}
			}
			continue
		}
		if codeFenceOpen {
			inCodeBlock[i] = true
			continue
		}

		if trimmed == "$$" {
			inMathBlock[i] = true
			mathBlockOpen = !mathBlockOpen
			continue
		}
		if strings.HasPrefix(trimmed, "$$") && strings.HasSuffix(trimmed, "$$") && len(trimmed) > 4 {
			inMathBlock[i] = true
			continue
		}
		if mathBlockOpen {
			inMathBlock[i] = true
		}
	}

	if codeFenceOpen {
		addViolation(codeFenceStartLine, RuleBlankLines, "unclosed fenced code block")
	}

	// 1. Parse all headings and check custom anchors, uniqueness, and blank lines around headings.
	var headings []parsedHeading
	anchorLines := make(map[string]int)

	for i, line := range lines {
		if inCodeBlock[i] {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			continue
		}
		rawMatch := rawHeadingRegex.FindStringSubmatch(line)
		if rawMatch == nil {
			addViolation(i+1, RuleHeadingAnchor, "malformed heading syntax %q", line)
			continue
		}

		level := len(rawMatch[1])
		if level > 5 {
			addViolation(i+1, RuleHeadingAnchor, "heading level H%d is too deep (maximum allowed level is H5 / #####)", level)
		}

		// Blank lines before/after headings
		if i == 0 {
			if i+1 < len(lines) && !isBlankLine(lines[i+1], opts.IsArtifact) {
				addViolation(i+1, RuleBlankLines, "document title heading on line 1 must be followed by a blank line")
			}
		} else {
			if !isBlankLine(lines[i-1], opts.IsArtifact) {
				addViolation(i+1, RuleBlankLines, "heading must be preceded by a blank line")
			}
			if i+1 >= len(lines) || !isBlankLine(lines[i+1], opts.IsArtifact) {
				addViolation(i+1, RuleBlankLines, "heading must be followed by a blank line")
			}
		}

		anchorMatch := anchoredHeadingRegex.FindStringSubmatch(line)
		if anchorMatch == nil {
			addViolation(i+1, RuleHeadingAnchor,
				"heading must have a custom anchor formatted as `%s <a id=\"anchor-text\" >Header text</a>` (got %q)",
				rawMatch[1], line)
			headings = append(headings, parsedHeading{
				line:      i + 1,
				level:     level,
				title:     strings.TrimSpace(rawMatch[2]),
				rawLine:   line,
				validForm: false,
			})
			continue
		}

		anchorID := anchorMatch[2]
		title := anchorMatch[3]
		if strings.TrimSpace(title) == "" {
			addViolation(i+1, RuleHeadingAnchor, "heading text inside <a>...</a> must not be empty")
		}
		if prevLine, dup := anchorLines[anchorID]; dup {
			addViolation(i+1, RuleHeadingAnchor, "duplicate heading anchor id %q (already defined on line %d)", anchorID, prevLine)
		} else {
			anchorLines[anchorID] = i + 1
		}

		headings = append(headings, parsedHeading{
			line:      i + 1,
			level:     level,
			anchorID:  anchorID,
			title:     title,
			rawLine:   line,
			validForm: true,
		})
	}

	// Check H1 Document Title on Line 1
	if len(headings) == 0 || headings[0].line != 1 || headings[0].level != 1 {
		addViolation(1, RuleDocumentTitle, "first line of the file must be the H1 document title (`# <a id=\"anchor-text\" >Title</a>`)")
	}
	for idx, h := range headings {
		if idx > 0 && h.level == 1 {
			addViolation(h.line, RuleDocumentTitle, "only one H1 heading (`# `) is allowed in the document (found additional H1 at line %d)", h.line)
		}
		if idx > 0 && h.level > headings[idx-1].level+1 {
			addViolation(h.line, RuleHeadingAnchor, "heading level jumped from H%d to H%d without intermediate level", headings[idx-1].level, h.level)
		}
	}

	// 2. Check Blank Lines around Lists, Horizontal Rules, and Lifecycle / Line Length / Reference Links
	checkLinesAndLists(lines, inCodeBlock, inMathBlock, anchorLines, opts, addViolation)

	// 3. Check Level 1 Headings (H2 in Markdown: Table of Contents + numbered sections)
	var h2Headings []int // indices into headings slice
	for idx, h := range headings {
		if h.level == 2 {
			h2Headings = append(h2Headings, idx)
		}
	}

	if len(h2Headings) == 0 {
		addViolation(1, RuleTableOfContents, "missing `## <a id=\"table-of-contents\" >Table of Contents</a>` and top-level sections")
		return violations
	}

	tocHeadingIdx := h2Headings[0]
	tocHeading := headings[tocHeadingIdx]
	if tocHeading.line != 3 || tocHeading.anchorID != "table-of-contents" || tocHeading.title != "Table of Contents" {
		addViolation(tocHeading.line, RuleTableOfContents,
			"first H2 heading must be `## <a id=\"table-of-contents\" >Table of Contents</a>` on line 3")
	}

	// Check Horizontal Separators (`---`) across the entire file.
	checkHorizontalSeparators(lines, inCodeBlock, headings, h2Headings, opts.IsArtifact, addViolation)

	// 4. Validate Top-Level Numbered Sections (order, numbering, anchors, required sections)
	var sections []sectionSpan
	lastCanonicalIdx := -1
	hasGoal := false
	hasProposal := false

	for sIdx := 1; sIdx < len(h2Headings); sIdx++ {
		hIdx := h2Headings[sIdx]
		h := headings[hIdx]

		endLine := len(lines)
		if sIdx+1 < len(h2Headings) {
			nextH2Line := headings[h2Headings[sIdx+1]].line
			endLine = nextH2Line - 1
			// Exclude the `---` separator and blank lines preceding next H2
			for endLine >= h.line && (isBlankLine(lines[endLine-1], opts.IsArtifact) || strings.TrimSpace(lines[endLine-1]) == "---") {
				endLine--
			}
		} else {
			for endLine >= h.line && isBlankLine(lines[endLine-1], opts.IsArtifact) {
				endLine--
			}
		}

		var subs []parsedHeading
		nextHIdx := len(headings)
		if sIdx+1 < len(h2Headings) {
			nextHIdx = h2Headings[sIdx+1]
		}
		for k := hIdx + 1; k < nextHIdx; k++ {
			subs = append(subs, headings[k])
		}

		numMatch := numberedSectionRegex.FindStringSubmatch(h.title)
		if numMatch == nil {
			addViolation(h.line, RuleTopLevelSections,
				"top-level section heading %q must be numbered (e.g. `%d. Goal`)", h.title, sIdx)
			continue
		}

		secNum, _ := strconv.Atoi(numMatch[1])
		secName := numMatch[2]
		if secNum != sIdx {
			addViolation(h.line, RuleTopLevelSections,
				"top-level section %q has number %d, expected sequential number %d", h.title, secNum, sIdx)
		}

		foundCanonical := -1
		for cIdx, c := range canonicalSections {
			if c.name == secName {
				foundCanonical = cIdx
				break
			}
		}

		if foundCanonical == -1 {
			allowedNames := make([]string, len(canonicalSections))
			for i, c := range canonicalSections {
				allowedNames[i] = c.name
			}
			addViolation(h.line, RuleTopLevelSections,
				"unknown top-level section name %q (allowed sections in order: %s)",
				secName, strings.Join(allowedNames, ", "))
			continue
		}

		if foundCanonical <= lastCanonicalIdx {
			addViolation(h.line, RuleTopLevelSections,
				"top-level section %q is out of canonical order (must appear after %q)",
				h.title, canonicalSections[lastCanonicalIdx].name)
		}
		lastCanonicalIdx = foundCanonical

		expectedAnchor := fmt.Sprintf("%d-%s", secNum, canonicalSections[foundCanonical].slug)
		if h.validForm && h.anchorID != expectedAnchor {
			addViolation(h.line, RuleTopLevelSections,
				"section %q anchor id must be %q (got %q)", h.title, expectedAnchor, h.anchorID)
		}

		if secName == "Goal" {
			hasGoal = true
		}
		if secName == "Proposal" {
			hasProposal = true
		}

		sections = append(sections, sectionSpan{
			headingIdx:    hIdx,
			heading:       h,
			canonicalIdx:  foundCanonical,
			canonicalName: secName,
			startLine:     h.line + 1,
			endLine:       endLine,
			subHeadings:   subs,
		})
	}

	if !hasGoal {
		addViolation(1, RuleTopLevelSections, "missing required top-level section `Goal`")
	}
	if !hasProposal {
		addViolation(1, RuleTopLevelSections, "missing required top-level section `Proposal`")
	}

	// 5. Validate Table of Contents entries against actual sections and Proposal design decisions
	tocEndLine := len(lines)
	if len(h2Headings) > 1 {
		tocEndLine = headings[h2Headings[1]].line - 1
		for tocEndLine >= tocHeading.line && (isBlankLine(lines[tocEndLine-1], opts.IsArtifact) || strings.TrimSpace(lines[tocEndLine-1]) == "---") {
			tocEndLine--
		}
	}
	validateTableOfContents(lines, tocHeading.line, tocEndLine, sections, opts.IsArtifact, addViolation)

	// 6. Validate each top-level section's internal structure
	for _, sec := range sections {
		validateSectionContent(lines, inCodeBlock, sec.startLine, sec.endLine, sec.heading, sec.canonicalName, sec.subHeadings, opts.IsArtifact, addViolation)
	}

	return violations
}

type sectionSpan struct {
	headingIdx    int
	heading       parsedHeading
	canonicalIdx  int
	canonicalName string
	startLine     int // 1-indexed line after heading
	endLine       int // 1-indexed line before next H2's `---` separator (or EOF)
	subHeadings   []parsedHeading
}

func isBlankLine(line string, isArtifact bool) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	if isArtifact && trimmed == "&nbsp;" {
		return true
	}
	return false
}

func isBlankOrEmptyBlockquote(line string, isArtifact bool) bool {
	if isBlankLine(line, isArtifact) {
		return true
	}
	trimmed := strings.TrimSpace(line)
	return trimmed == ">"
}

func checkHorizontalSeparators(
	lines []string,
	inCodeBlock []bool,
	headings []parsedHeading,
	h2Headings []int,
	isArtifact bool,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	// Identify all `---` lines outside fenced code blocks
	var hrLines []int
	for i, line := range lines {
		if inCodeBlock[i] {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "***" || trimmed == "___" {
			addViolation(i+1, RuleHorizontalSeparator, "use `---` for horizontal separators, not %q", trimmed)
			continue
		}
		if trimmed == "---" {
			hrLines = append(hrLines, i+1)
			if i == 0 || !isBlankLine(lines[i-1], isArtifact) {
				addViolation(i+1, RuleBlankLines, "horizontal separator `---` must have a blank line before it")
			}
			if i+1 >= len(lines) || !isBlankLine(lines[i+1], isArtifact) {
				addViolation(i+1, RuleBlankLines, "horizontal separator `---` must have a blank line after it")
			}
		}
	}

	// Expected `---` locations: exactly one `---` immediately preceding each numbered H2 section (sIdx >= 1).
	expectedHRForH2 := make(map[int]int) // hrLine -> h2Line
	for sIdx := 1; sIdx < len(h2Headings); sIdx++ {
		h2 := headings[h2Headings[sIdx]]
		// Walk backwards from h2.line - 1 skipping blank lines
		p := h2.line - 1
		for p >= 1 && isBlankLine(lines[p-1], isArtifact) {
			p--
		}
		if p >= 1 && !inCodeBlock[p-1] && strings.TrimSpace(lines[p-1]) == "---" {
			expectedHRForH2[p] = h2.line
		} else {
			addViolation(h2.line, RuleHorizontalSeparator,
				"Level 1 section heading %q must be preceded by a `---` horizontal separator", h2.title)
		}
	}

	for _, hrLine := range hrLines {
		if _, ok := expectedHRForH2[hrLine]; !ok {
			addViolation(hrLine, RuleHorizontalSeparator,
				"horizontal separator `---` is only allowed between Level 1 (`##`) headings")
		}
	}
}

func validateTableOfContents(
	lines []string,
	tocHeadingLine int,
	tocEndLine int,
	sections []sectionSpan,
	isArtifact bool,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	type expectedTocItem struct {
		indent   int
		text     string
		anchorID string
	}

	var expected []expectedTocItem
	for _, sec := range sections {
		expected = append(expected, expectedTocItem{
			indent:   0,
			text:     sec.heading.title,
			anchorID: sec.heading.anchorID,
		})
		if sec.canonicalName == "Proposal" {
			for _, sub := range sec.subHeadings {
				if sub.level == 3 {
					expected = append(expected, expectedTocItem{
						indent:   2,
						text:     sub.title,
						anchorID: sub.anchorID,
					})
				}
			}
		}
	}

	type actualTocItem struct {
		line     int
		indent   int
		text     string
		anchorID string
	}

	var actual []actualTocItem
	for lineNum := tocHeadingLine + 1; lineNum <= tocEndLine; lineNum++ {
		line := lines[lineNum-1]
		if isBlankLine(line, isArtifact) {
			continue
		}
		if m := tocTopItemRegex.FindStringSubmatch(line); m != nil {
			actual = append(actual, actualTocItem{
				line:     lineNum,
				indent:   0,
				text:     m[1],
				anchorID: m[2],
			})
		} else if m := tocSubItemRegex.FindStringSubmatch(line); m != nil {
			actual = append(actual, actualTocItem{
				line:     lineNum,
				indent:   2,
				text:     m[1],
				anchorID: m[2],
			})
		} else {
			addViolation(lineNum, RuleTableOfContents,
				"invalid Table of Contents line %q (expected `- [Section](#anchor)` or `  - [Design Decision](#anchor)`)", line)
		}
	}

	if len(actual) == 0 {
		addViolation(tocHeadingLine, RuleTableOfContents, "Table of Contents is empty")
		return
	}

	maxLen := len(expected)
	if len(actual) > maxLen {
		maxLen = len(actual)
	}
	for i := 0; i < maxLen; i++ {
		if i >= len(expected) {
			item := actual[i]
			addViolation(item.line, RuleTableOfContents,
				"unexpected extra Table of Contents entry `[%s](#%s)` (only Level 1 sections and Proposal design decision headings belong in the Table of Contents)",
				item.text, item.anchorID)
			continue
		}
		if i >= len(actual) {
			exp := expected[i]
			prefix := "- "
			if exp.indent == 2 {
				prefix = "  - "
			}
			addViolation(tocHeadingLine, RuleTableOfContents,
				"missing Table of Contents entry `%s[%s](#%s)`", prefix, exp.text, exp.anchorID)
			continue
		}
		act := actual[i]
		exp := expected[i]
		if act.indent != exp.indent || act.anchorID != exp.anchorID || act.text != exp.text {
			expPrefix := "- "
			if exp.indent == 2 {
				expPrefix = "  - "
			}
			actPrefix := "- "
			if act.indent == 2 {
				actPrefix = "  - "
			}
			addViolation(act.line, RuleTableOfContents,
				"Table of Contents entry mismatch: got `%s[%s](#%s)`, expected `%s[%s](#%s)`",
				actPrefix, act.text, act.anchorID, expPrefix, exp.text, exp.anchorID)
		}
	}
}

func checkLinesAndLists(
	lines []string,
	inCodeBlock []bool,
	inMathBlock []bool,
	anchorLines map[string]int,
	opts Options,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	inList := false

	for i, line := range lines {
		lineNum := i + 1
		if inCodeBlock[i] {
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Check &nbsp; usage based on lifecycle mode
		if !opts.IsArtifact {
			withoutInlineCode := inlineCodeRegex.ReplaceAllString(line, "")
			if strings.Contains(withoutInlineCode, "&nbsp;") {
				addViolation(lineNum, RuleLifecycleFormat,
					"committed plan must not contain `&nbsp;` spacing (strip `&nbsp;` when moving from artifact to docs/plans/)")
			}
		}

		// Check markdown links and bare URLs
		checkLineLinks(line, lineNum, anchorLines, opts, addViolation)

		// Check 80-character line wrapping
		if !inMathBlock[i] && !strings.HasPrefix(line, "#") && !strings.HasPrefix(trimmed, "|") &&
			tocTopItemRegex.FindStringSubmatch(line) == nil && tocSubItemRegex.FindStringSubmatch(line) == nil {
			if utf8.RuneCountInString(line) > 80 && canLineWrapEarlier(line) {
				addViolation(lineNum, RuleLineLength,
					"line is %d characters long (lines must wrap at 80 characters whenever possible)",
					utf8.RuneCountInString(line))
			}
		}

		// List blank line tracking
		isListItem := isListLine(line)
		if isListItem {
			if !inList {
				// Starting a new list block: previous line must be blank (or `>` if blockquote list)
				if i > 0 && !isBlankOrEmptyBlockquote(lines[i-1], opts.IsArtifact) {
					addViolation(lineNum, RuleBlankLines,
						"list must be preceded by a blank line")
				}
				inList = true
			}
			continue
		}

		if inList {
			if isBlankOrEmptyBlockquote(line, opts.IsArtifact) {
				// Look ahead to see if the list continues after this blank line
				nextNonBlank := i + 1
				for nextNonBlank < len(lines) && isBlankOrEmptyBlockquote(lines[nextNonBlank], opts.IsArtifact) {
					nextNonBlank++
				}
				if nextNonBlank < len(lines) && (isListLine(lines[nextNonBlank]) || isIndentedContinuation(lines[nextNonBlank])) {
					continue
				}
				inList = false
				continue
			}

			// Non-blank non-list-item line while inList is true:
			// Is it an indented continuation of the current list item (or blockquote list continuation)?
			if isIndentedContinuation(line) || isBlockquoteListContinuation(line) {
				continue
			}

			// Otherwise, a non-indented paragraph/heading/etc. started immediately after a list without a blank line!
			addViolation(lineNum, RuleBlankLines,
				"list must be followed by a blank line before %q", trimmed)
			inList = false
		}
	}
}

func isListLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "---" || trimmed == "***" || trimmed == "___" {
		return false
	}
	// Strip leading blockquote marker `>` if list is inside a blockquote
	checkLine := line
	if strings.HasPrefix(trimmed, ">") {
		checkLine = strings.TrimPrefix(trimmed, ">")
	}
	return unorderedListRegex.MatchString(checkLine) || orderedListRegex.MatchString(checkLine)
}

func isIndentedContinuation(line string) bool {
	return strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")
}

func isBlockquoteListContinuation(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, ">   ") || strings.HasPrefix(trimmed, ">  ")
}

func canLineWrapEarlier(line string) bool {
	// Strip leading indentation, blockquote markers, and list markers
	s := strings.TrimLeft(line, " \t")
	for strings.HasPrefix(s, ">") {
		s = strings.TrimLeft(s[1:], " \t")
	}
	if m := unorderedListRegex.FindStringSubmatch(s); m != nil {
		s = m[2]
	} else if m := orderedListRegex.FindStringSubmatch(s); m != nil {
		s = m[3]
	}
	s = strings.TrimSpace(s)

	// Replace inline links `[...](...)`, inline code `` `...` ``, and inline math `$...$` with solid tokens
	masked := inlineCodeRegex.ReplaceAllStringFunc(s, func(m string) string {
		return strings.Repeat("X", len(m))
	})
	masked = markdownLinkRegex.ReplaceAllStringFunc(masked, func(m string) string {
		return strings.Repeat("L", len(m))
	})
	masked = inlineMathRegex.ReplaceAllStringFunc(masked, func(m string) string {
		return strings.Repeat("M", len(m))
	})

	// If there is still a space outside of single unbreakable tokens, the line can wrap!
	return strings.Contains(masked, " ")
}

func checkLineLinks(
	line string,
	lineNum int,
	anchorLines map[string]int,
	opts Options,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	// Mask inline code spans so backticked examples don't trigger link violations
	withoutCode := inlineCodeRegex.ReplaceAllStringFunc(line, func(m string) string {
		return strings.Repeat(" ", len(m))
	})

	matches := markdownLinkRegex.FindAllStringSubmatchIndex(withoutCode, -1)
	for _, idx := range matches {
		isImage := line[idx[0]] == '!'
		text := strings.TrimSpace(line[idx[2]:idx[3]])
		target := strings.TrimSpace(line[idx[4]:idx[5]])

		if !isImage {
			if text == "" {
				addViolation(lineNum, RuleReferenceLinks, "markdown link `[] (%s)` has empty anchor text", target)
			} else if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") || strings.HasPrefix(text, "file://") {
				addViolation(lineNum, RuleReferenceLinks,
					"markdown link `[%s](%s)` must use descriptive anchor text rather than a raw URL", text, target)
			} else {
				lowerText := strings.ToLower(text)
				if lowerText == "link" || lowerText == "here" || lowerText == "click here" || lowerText == "this" || lowerText == "url" {
					addViolation(lineNum, RuleReferenceLinks,
						"markdown link `[%s](%s)` must use descriptive anchor text instead of generic %q", text, target, text)
				}
			}
		}

		// Validate target scheme and path resolution
		if strings.HasPrefix(target, "#") {
			anchor := strings.TrimPrefix(target, "#")
			if _, ok := anchorLines[anchor]; !ok {
				addViolation(lineNum, RuleReferenceLinks,
					"internal anchor link `[%s](%s)` references non-existent heading anchor %q", text, target, anchor)
			}
			continue
		}

		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") ||
			strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "conversation://") {
			continue
		}

		if !opts.IsArtifact {
			if strings.HasPrefix(target, "file://") {
				addViolation(lineNum, RuleLifecycleFormat,
					"committed plan contains `file://` link `[%s](%s)` (convert to repository-relative path)", text, target)
				continue
			}
			if strings.HasPrefix(target, "/") {
				addViolation(lineNum, RuleLifecycleFormat,
					"committed plan contains absolute path link `[%s](%s)` (use repository-relative path)", text, target)
				continue
			}
			// Relative path check
			if !opts.SkipFileExistenceCheck && opts.FilePath != "" {
				pathPart := target
				if hashIdx := strings.IndexByte(pathPart, '#'); hashIdx != -1 {
					pathPart = pathPart[:hashIdx]
				}
				if pathPart != "" {
					resolved := filepath.Join(filepath.Dir(opts.FilePath), pathPart)
					if _, err := os.Stat(resolved); err != nil {
						addViolation(lineNum, RuleReferenceLinks,
							"relative file link `[%s](%s)` does not resolve to an existing file (%s)", text, target, resolved)
					}
				}
			}
		} else {
			// In artifact mode, local file links must use file:/// scheme
			if !strings.HasPrefix(target, "file:///") {
				addViolation(lineNum, RuleLifecycleFormat,
					"artifact plan internal link `[%s](%s)` must use the `file:///` scheme for editor clickability", text, target)
			} else if !opts.SkipFileExistenceCheck {
				pathPart := strings.TrimPrefix(target, "file://")
				if hashIdx := strings.IndexByte(pathPart, '#'); hashIdx != -1 {
					pathPart = pathPart[:hashIdx]
				}
				if pathPart != "" {
					if _, err := os.Stat(pathPart); err != nil {
						addViolation(lineNum, RuleReferenceLinks,
							"artifact file link `[%s](%s)` does not resolve to an existing file (%s)", text, target, pathPart)
					}
				}
			}
		}
	}

	// Check for bare URLs outside markdown links and inline code
	withoutLinks := markdownLinkRegex.ReplaceAllStringFunc(withoutCode, func(m string) string {
		return strings.Repeat(" ", len(m))
	})
	if bareMatch := bareURLRegex.FindString(withoutLinks); bareMatch != "" {
		addViolation(lineNum, RuleReferenceLinks,
			"bare URL %q found in prose (format as a clickable link `[descriptive text](%s)`)", bareMatch, bareMatch)
	}
}

func validateSectionContent(
	lines []string,
	inCodeBlock []bool,
	startLine int,
	endLine int,
	secHeading parsedHeading,
	canonicalName string,
	subHeadings []parsedHeading,
	isArtifact bool,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	// Check non-empty & non-placeholder content
	var nonBlankLines []string
	for ln := startLine; ln <= endLine; ln++ {
		if !isBlankLine(lines[ln-1], isArtifact) {
			nonBlankLines = append(nonBlankLines, strings.TrimSpace(lines[ln-1]))
		}
	}

	if len(nonBlankLines) == 0 {
		addViolation(secHeading.line, RuleEmptySection,
			"section %q is empty (omit empty optional sections completely)", secHeading.title)
		return
	}

	if len(nonBlankLines) == 1 && placeholderTextRegex.MatchString(nonBlankLines[0]) {
		addViolation(secHeading.line, RuleEmptySection,
			"section %q contains placeholder text %q (omit empty optional sections completely instead of using placeholders)",
			secHeading.title, nonBlankLines[0])
		return
	}

	switch canonicalName {
	case "Background", "Goal":
		for _, sub := range subHeadings {
			addViolation(sub.line, RuleTopLevelSections,
				"section %q must not contain sub-headings (found %q)", secHeading.title, sub.rawLine)
		}

	case "Proposal":
		validateProposalSection(lines, inCodeBlock, startLine, endLine, secHeading, subHeadings, isArtifact, addViolation)

	case "Alternatives considered":
		validateTopLevelAlternativesSection(lines, inCodeBlock, startLine, endLine, secHeading, subHeadings, isArtifact, addViolation)

	case "Assumptions":
		validateNumberedListSection(lines, inCodeBlock, startLine, endLine, secHeading, subHeadings, RuleAssumptionsFormat, isArtifact, addViolation)

	case "Future work":
		validateFutureWorkSection(lines, startLine, endLine, secHeading, subHeadings, isArtifact, addViolation)

	case "Open questions":
		validateNumberedListSection(lines, inCodeBlock, startLine, endLine, secHeading, subHeadings, RuleOpenQuestionsFormat, isArtifact, addViolation)
	}
}

func validateProposalSection(
	lines []string,
	inCodeBlock []bool,
	startLine int,
	endLine int,
	secHeading parsedHeading,
	subHeadings []parsedHeading,
	isArtifact bool,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	// Find first non-blank line in Proposal
	firstLine := startLine
	for firstLine <= endLine && isBlankLine(lines[firstLine-1], isArtifact) {
		firstLine++
	}
	if firstLine > endLine {
		return
	}

	// 1. Must start with `> **TL;DR:**` blockquote
	if !strings.HasPrefix(strings.TrimSpace(lines[firstLine-1]), "> **TL;DR:**") {
		addViolation(firstLine, RuleProposalTLDR,
			"Proposal section must start immediately with a `> **TL;DR:** <explanation>` blockquote")
	} else {
		tldrEndLine := firstLine
		for tldrEndLine <= endLine && strings.HasPrefix(strings.TrimSpace(lines[tldrEndLine-1]), ">") {
			tldrEndLine++
		}
		tldrEndLine--
		validateTLDRBlockquote(lines[firstLine-1:tldrEndLine], firstLine, addViolation)

		// 2. Check high-level concept explanation paragraph between TL;DR and first `###` design decision
		firstDecisionLine := endLine + 1
		for _, sub := range subHeadings {
			if sub.level == 3 {
				firstDecisionLine = sub.line
				break
			}
		}

		hasOverviewProse := false
		for ln := tldrEndLine + 1; ln < firstDecisionLine; ln++ {
			if inCodeBlock[ln-1] || isBlankLine(lines[ln-1], isArtifact) {
				continue
			}
			trimmed := strings.TrimSpace(lines[ln-1])
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			hasOverviewProse = true
			break
		}
		if !hasOverviewProse {
			addViolation(tldrEndLine, RuleProposalOverview,
				"Proposal section must include a high-level concept explanation paragraph immediately following the `> **TL;DR:**` blockquote and before the first `###` design decision")
		}
	}

	// 3. Validate `###` Design Decisions and nested `#### Alternatives considered` -> `##### Alternative Name`
	var h3Indices []int
	for idx, sub := range subHeadings {
		if sub.level == 3 {
			h3Indices = append(h3Indices, idx)
		}
	}

	if len(h3Indices) == 0 {
		addViolation(secHeading.line, RuleProposalDecisions,
			"Proposal section must contain at least one `### <a id=\"custom-anchor-text\" >Design Decision</a>` sub-heading")
		return
	}

	// Ensure no sub-headings appear before the first H3 design decision
	if h3Indices[0] > 0 {
		badSub := subHeadings[0]
		addViolation(badSub.line, RuleProposalDecisions,
			"unexpected H%d sub-heading %q before first `###` design decision in Proposal", badSub.level, badSub.title)
	}

	for dIdx, h3Pos := range h3Indices {
		h3 := subHeadings[h3Pos]
		nextH3Pos := len(subHeadings)
		decisionEndLine := endLine
		if dIdx+1 < len(h3Indices) {
			nextH3Pos = h3Indices[dIdx+1]
			decisionEndLine = subHeadings[nextH3Pos].line - 1
		}

		children := subHeadings[h3Pos+1 : nextH3Pos]
		bodyEndLine := decisionEndLine
		if len(children) > 0 {
			bodyEndLine = children[0].line - 1
		}

		// Verify design decision has non-empty explanation before any H4
		hasDecisionBody := false
		for ln := h3.line + 1; ln <= bodyEndLine; ln++ {
			if !isBlankLine(lines[ln-1], isArtifact) {
				hasDecisionBody = true
				break
			}
		}
		if !hasDecisionBody {
			addViolation(h3.line, RuleProposalDecisions,
				"design decision %q must have a proposal explanation directly under it", h3.title)
		}

		if len(children) == 0 {
			continue
		}

		// The only allowed H4 under a design decision is `#### <a id="..." >Alternatives considered</a>`
		var h4Positions []int
		for cIdx, child := range children {
			if child.level == 4 {
				h4Positions = append(h4Positions, cIdx)
			}
		}

		if len(h4Positions) == 0 || h4Positions[0] != 0 {
			addViolation(children[0].line, RuleDecisionAlternatives,
				"sub-heading under design decision %q must be `#### <a id=\"...\" >Alternatives considered</a>` (got %q)",
				h3.title, children[0].rawLine)
			continue
		}

		if len(h4Positions) > 1 {
			secondH4 := children[h4Positions[1]]
			addViolation(secondH4.line, RuleProposalDecisions,
				"only one `#### <a id=\"...\" >Alternatives considered</a>` section is allowed under design decision %q (found extra H4 %q)",
				h3.title, secondH4.title)
		}

		altH4 := children[0]
		if altH4.title != "Alternatives considered" {
			addViolation(altH4.line, RuleProposalDecisions,
				"only `#### <a id=\"...\" >Alternatives considered</a>` is allowed as an H4 sub-heading under a design decision (got %q)",
				altH4.title)
		}

		h5List := children[1:]
		if len(h5List) == 0 {
			addViolation(altH4.line, RuleDecisionAlternatives,
				"`#### <a id=%q >Alternatives considered</a>` under %q must contain at least one `##### <a id=\"custom-anchor-text\" >Alternative Name</a>` sub-heading",
				altH4.anchorID, h3.title)
			continue
		}

		// Ensure no bare content between `#### Alternatives considered` and first `##### Alternative Name`
		for ln := altH4.line + 1; ln < h5List[0].line; ln++ {
			if !isBlankLine(lines[ln-1], isArtifact) {
				addViolation(ln, RuleDecisionAlternatives,
					"unexpected content directly under `#### Alternatives considered` before first `##### <a id=\"...\" >Alternative Name</a>` sub-heading")
				break
			}
		}

		for aIdx, altH5 := range h5List {
			if altH5.level != 5 {
				addViolation(altH5.line, RuleDecisionAlternatives,
					"expected `##### <a id=\"...\" >Alternative Name</a>` (H5) under `#### Alternatives considered`, got H%d %q",
					altH5.level, altH5.rawLine)
				continue
			}
			altEnd := decisionEndLine
			if aIdx+1 < len(h5List) {
				altEnd = h5List[aIdx+1].line - 1
			}
			validateAlternativeItem(lines, inCodeBlock, altH5, altH5.line+1, altEnd, RuleDecisionAlternatives, isArtifact, addViolation)
		}
	}
}

func validateTLDRBlockquote(
	tldrLines []string,
	startLine int,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	// Strip leading `>` and optional single space from each line
	stripped := make([]string, len(tldrLines))
	for i, raw := range tldrLines {
		s := strings.TrimSpace(raw)
		s = strings.TrimPrefix(s, ">")
		s = strings.TrimPrefix(s, " ")
		stripped[i] = s
	}

	// Remove `**TL;DR:**` prefix on the first line
	first := strings.TrimSpace(strings.TrimPrefix(stripped[0], "**TL;DR:**"))
	stripped[0] = first

	type tldrStatement struct {
		line int
		text string
	}

	var bulletStatements []tldrStatement
	var paragraphLines []string
	var currentBullet *tldrStatement

	for i, s := range stripped {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			if currentBullet != nil {
				bulletStatements = append(bulletStatements, *currentBullet)
				currentBullet = nil
			}
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if currentBullet != nil {
				bulletStatements = append(bulletStatements, *currentBullet)
			}
			currentBullet = &tldrStatement{
				line: startLine + i,
				text: strings.TrimSpace(trimmed[2:]),
			}
		} else if currentBullet != nil {
			currentBullet.text += " " + trimmed
		} else {
			paragraphLines = append(paragraphLines, trimmed)
		}
	}
	if currentBullet != nil {
		bulletStatements = append(bulletStatements, *currentBullet)
	}

	var statements []tldrStatement
	if len(bulletStatements) > 0 {
		// If there is also a lead-in line before the bullets, check if it's a statement or header
		if len(paragraphLines) > 0 {
			leadIn := strings.Join(paragraphLines, " ")
			// Count lead-in if it is a standalone sentence/statement; always check its word count
			if wc := countStatementWords(leadIn); wc > 20 {
				addViolation(startLine, RuleProposalTLDR,
					"TL;DR lead-in statement has %d words (maximum is 20 words per statement): %q", wc, leadIn)
			}
		}
		statements = bulletStatements
	} else if len(paragraphLines) > 0 {
		fullPara := strings.Join(paragraphLines, " ")
		for _, sent := range splitSentences(fullPara) {
			statements = append(statements, tldrStatement{
				line: startLine,
				text: sent,
			})
		}
	}

	if len(statements) == 0 {
		addViolation(startLine, RuleProposalTLDR, "`> **TL;DR:**` blockquote must contain an explanation (1 to 5 statements)")
		return
	}

	if len(statements) > 5 {
		addViolation(startLine, RuleProposalTLDR,
			"`> **TL;DR:**` blockquote has %d statements (maximum allowed is 5 statements)", len(statements))
	}

	for _, st := range statements {
		wc := countStatementWords(st.text)
		if wc > 20 {
			addViolation(st.line, RuleProposalTLDR,
				"TL;DR statement has %d words (maximum allowed is 20 words per statement): %q", wc, st.text)
		}
	}
}

func countStatementWords(s string) int {
	// Replace markdown links `[text](url)` with `text`
	cleaned := markdownLinkRegex.ReplaceAllString(s, "$1")
	// Collapse inline LaTeX math `$...$` into a single word token
	cleaned = inlineMathRegex.ReplaceAllString(cleaned, "MATH")
	// Strip markdown emphasis/code delimiters
	cleaned = strings.NewReplacer("**", "", "__", "", "`", "").Replace(cleaned)
	fields := strings.Fields(cleaned)
	count := 0
	for _, f := range fields {
		trimmed := strings.Trim(f, ".,;:!?()[]{}\"'+-/=")
		if trimmed != "" {
			count++
		}
	}
	return count
}

func splitSentences(paragraph string) []string {
	var sentences []string
	var cur strings.Builder
	words := strings.Fields(paragraph)
	for i, w := range words {
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(w)
		isAbbrev := strings.HasSuffix(w, "e.g.") || strings.HasSuffix(w, "i.e.") || strings.HasSuffix(w, "vs.") || strings.HasSuffix(w, "etc.")
		if !isAbbrev && (strings.HasSuffix(w, ".") || strings.HasSuffix(w, "!") || strings.HasSuffix(w, "?")) {
			sentences = append(sentences, strings.TrimSpace(cur.String()))
			cur.Reset()
		} else if i == len(words)-1 && cur.Len() > 0 {
			sentences = append(sentences, strings.TrimSpace(cur.String()))
			cur.Reset()
		}
	}
	return sentences
}

func validateTopLevelAlternativesSection(
	lines []string,
	inCodeBlock []bool,
	startLine int,
	endLine int,
	secHeading parsedHeading,
	subHeadings []parsedHeading,
	isArtifact bool,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	if len(subHeadings) == 0 {
		addViolation(secHeading.line, RuleTopLevelAlternatives,
			"`%s` section must contain at least one `### <a id=\"custom-anchor-text\" >Alternative Name</a>` sub-heading",
			secHeading.title)
		return
	}

	for ln := startLine; ln < subHeadings[0].line; ln++ {
		if !isBlankLine(lines[ln-1], isArtifact) {
			addViolation(ln, RuleTopLevelAlternatives,
				"unexpected content in `%s` before first `### <a id=\"...\" >Alternative Name</a>` sub-heading",
				secHeading.title)
			break
		}
	}

	for idx, sub := range subHeadings {
		if sub.level != 3 {
			addViolation(sub.line, RuleTopLevelAlternatives,
				"expected `### <a id=\"...\" >Alternative Name</a>` (H3) in `%s`, got H%d %q",
				secHeading.title, sub.level, sub.rawLine)
			continue
		}
		altEnd := endLine
		if idx+1 < len(subHeadings) {
			altEnd = subHeadings[idx+1].line - 1
		}
		validateAlternativeItem(lines, inCodeBlock, sub, sub.line+1, altEnd, RuleTopLevelAlternatives, isArtifact, addViolation)
	}
}

func validateAlternativeItem(
	lines []string,
	inCodeBlock []bool,
	altHeading parsedHeading,
	startLine int,
	endLine int,
	rule RuleID,
	isArtifact bool,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	// Parse blocks separated by blank lines within [startLine, endLine]
	type textBlock struct {
		startLine int
		lines     []string
	}

	var blocks []textBlock
	var cur *textBlock

	for ln := startLine; ln <= endLine; ln++ {
		line := lines[ln-1]
		if !inCodeBlock[ln-1] && isBlankLine(line, isArtifact) {
			if cur != nil {
				blocks = append(blocks, *cur)
				cur = nil
			}
			continue
		}
		if cur == nil {
			cur = &textBlock{startLine: ln}
		}
		cur.lines = append(cur.lines, line)
	}
	if cur != nil {
		blocks = append(blocks, *cur)
	}

	if len(blocks) == 0 {
		addViolation(altHeading.line, rule,
			"alternative %q is empty (must include an unbulleted ~2 sentence summary, `**Pros:**` with bullets, and `**Cons:**` with bullets)",
			altHeading.title)
		return
	}

	// Expected 5 blocks:
	// Block 0: Unbulleted ~2 sentence summary paragraph
	// Block 1: `**Pros:**`
	// Block 2: Bulleted list of pros
	// Block 3: `**Cons:**`
	// Block 4: Bulleted list of cons

	summaryBlock := blocks[0]
	firstSummaryLine := strings.TrimSpace(summaryBlock.lines[0])
	if isListLine(summaryBlock.lines[0]) || firstSummaryLine == "**Pros:**" || firstSummaryLine == "**Cons:**" ||
		strings.HasPrefix(firstSummaryLine, "**Pros:**") || strings.HasPrefix(firstSummaryLine, "**Cons:**") {
		addViolation(summaryBlock.startLine, rule,
			"alternative %q must begin with an unbulleted ~2 sentence summary paragraph before `**Pros:**`",
			altHeading.title)
		return
	}
	for offset, l := range summaryBlock.lines {
		if isListLine(l) {
			addViolation(summaryBlock.startLine+offset, rule,
				"alternative %q summary must be an unbulleted paragraph", altHeading.title)
			break
		}
	}

	if len(blocks) < 5 {
		addViolation(altHeading.line, rule,
			"alternative %q is incomplete: must contain an unbulleted summary paragraph, an unbulleted `**Pros:**` label followed by a bulleted list, and an unbulleted `**Cons:**` label followed by a bulleted list",
			altHeading.title)
		return
	}

	if len(blocks) > 5 {
		addViolation(blocks[5].startLine, rule,
			"alternative %q has unexpected extra content or multiple paragraphs (must strictly consist of 1 summary paragraph, `**Pros:**` + bulleted list, and `**Cons:**` + bulleted list)",
			altHeading.title)
	}

	// Check Block 1: `**Pros:**`
	prosLabelBlock := blocks[1]
	if len(prosLabelBlock.lines) != 1 || strings.TrimSpace(prosLabelBlock.lines[0]) != "**Pros:**" {
		addViolation(prosLabelBlock.startLine, rule,
			"alternative %q must have an unbulleted `**Pros:**` label on its own line after the summary paragraph (got %q)",
			altHeading.title, strings.TrimSpace(prosLabelBlock.lines[0]))
	}

	// Check Block 2: Bulleted list of pros
	prosListBlock := blocks[2]
	if !isCleanBulletListBlock(prosListBlock.lines) {
		addViolation(prosListBlock.startLine, rule,
			"alternative %q `**Pros:**` must be followed by a bulleted list (`- ...`)", altHeading.title)
	}

	// Check Block 3: `**Cons:**`
	consLabelBlock := blocks[3]
	if len(consLabelBlock.lines) != 1 || strings.TrimSpace(consLabelBlock.lines[0]) != "**Cons:**" {
		addViolation(consLabelBlock.startLine, rule,
			"alternative %q must have an unbulleted `**Cons:**` label on its own line after the pros list (got %q)",
			altHeading.title, strings.TrimSpace(consLabelBlock.lines[0]))
	}

	// Check Block 4: Bulleted list of cons
	consListBlock := blocks[4]
	if !isCleanBulletListBlock(consListBlock.lines) {
		addViolation(consListBlock.startLine, rule,
			"alternative %q `**Cons:**` must be followed by a bulleted list (`- ...`)", altHeading.title)
	}
}

func isCleanBulletListBlock(blockLines []string) bool {
	if len(blockLines) == 0 {
		return false
	}
	if !strings.HasPrefix(strings.TrimSpace(blockLines[0]), "- ") && !strings.HasPrefix(strings.TrimSpace(blockLines[0]), "* ") {
		return false
	}
	for _, l := range blockLines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || isIndentedContinuation(l) {
			continue
		}
		return false
	}
	return true
}

func validateNumberedListSection(
	lines []string,
	inCodeBlock []bool,
	startLine int,
	endLine int,
	secHeading parsedHeading,
	subHeadings []parsedHeading,
	rule RuleID,
	isArtifact bool,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	for _, sub := range subHeadings {
		addViolation(sub.line, rule,
			"`%s` section must be a numbered list and must not contain sub-headings (found %q)",
			secHeading.title, sub.rawLine)
	}

	expectedNum := 1
	inItem := false
	for ln := startLine; ln <= endLine; ln++ {
		if inCodeBlock[ln-1] {
			continue
		}
		line := lines[ln-1]
		if isBlankLine(line, isArtifact) {
			continue
		}
		if m := orderedListRegex.FindStringSubmatch(line); m != nil && m[1] == "" {
			num, _ := strconv.Atoi(m[2])
			if num != expectedNum {
				addViolation(ln, rule,
					"numbered list item in `%s` has number %d, expected sequential number %d",
					secHeading.title, num, expectedNum)
			}
			expectedNum++
			inItem = true
			continue
		}
		if inItem && isIndentedContinuation(line) {
			continue
		}
		addViolation(ln, rule,
			"`%s` section must consist exclusively of a numbered list (`1. ...`, `2. ...`), found %q",
			secHeading.title, strings.TrimSpace(line))
	}

	if expectedNum == 1 {
		addViolation(secHeading.line, rule,
			"`%s` section must contain at least one numbered list item (`1. ...`)", secHeading.title)
	}
}

func validateFutureWorkSection(
	lines []string,
	startLine int,
	endLine int,
	secHeading parsedHeading,
	subHeadings []parsedHeading,
	isArtifact bool,
	addViolation func(line int, rule RuleID, format string, args ...any),
) {
	if len(subHeadings) == 0 {
		addViolation(secHeading.line, RuleFutureWorkFormat,
			"`%s` section must contain 1 `### <a id=\"custom-anchor-text\" >Idea</a>` sub-heading per idea",
			secHeading.title)
		return
	}

	for ln := startLine; ln < subHeadings[0].line; ln++ {
		if !isBlankLine(lines[ln-1], isArtifact) {
			addViolation(ln, RuleFutureWorkFormat,
				"unexpected content in `%s` before first `### <a id=\"...\" >Idea</a>` sub-heading",
				secHeading.title)
			break
		}
	}

	for idx, sub := range subHeadings {
		if sub.level != 3 {
			addViolation(sub.line, RuleFutureWorkFormat,
				"`%s` section only permits `###` (H3) idea sub-headings (found H%d %q)",
				secHeading.title, sub.level, sub.rawLine)
			continue
		}
		ideaEnd := endLine
		if idx+1 < len(subHeadings) {
			ideaEnd = subHeadings[idx+1].line - 1
		}
		hasContent := false
		for ln := sub.line + 1; ln <= ideaEnd; ln++ {
			if !isBlankLine(lines[ln-1], isArtifact) {
				hasContent = true
				break
			}
		}
		if !hasContent {
			addViolation(sub.line, RuleFutureWorkFormat,
				"future work idea %q must have an explanation under it", sub.title)
		}
	}
}
