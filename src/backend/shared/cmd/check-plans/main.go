package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"quadsmith/shared/pkg/planlint"
)

func main() {
	isArtifact := flag.Bool("artifact", false, "Lint files as draft session artifacts (allows &nbsp; spacing and requires file:/// internal links)")
	dirFlag := flag.String("dir", "docs/plans", "Directory containing committed plan documents to lint when no file arguments are provided")
	flag.Parse()

	args := flag.Args()
	var checkedFiles []string
	var allViolations []planlint.Violation

	if len(args) > 0 {
		for _, arg := range args {
			info, err := os.Stat(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Plan Linter Error: cannot access %s: %v\n", arg, err)
				os.Exit(1)
			}
			if info.IsDir() {
				files, violations, err := planlint.LintDirectory(arg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Plan Linter Error: %v\n", err)
					os.Exit(1)
				}
				checkedFiles = append(checkedFiles, files...)
				allViolations = append(allViolations, violations...)
				continue
			}

			artifactMode := *isArtifact || isArtifactPath(arg)
			checkedFiles = append(checkedFiles, arg)
			violations, err := planlint.LintFile(arg, planlint.Options{
				FilePath:   arg,
				IsArtifact: artifactMode,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Plan Linter Error: %v\n", err)
				os.Exit(1)
			}
			allViolations = append(allViolations, violations...)
		}
	} else {
		files, violations, err := planlint.LintDirectory(*dirFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Plan Linter Error: %v\n", err)
			os.Exit(1)
		}
		checkedFiles = files
		allViolations = violations
	}

	if len(allViolations) > 0 {
		fmt.Fprintf(os.Stderr, "❌ Plan Document Linter Failed (%d violation(s) across %d file(s)):\n", len(allViolations), len(checkedFiles))
		for _, v := range allViolations {
			fmt.Fprintf(os.Stderr, "  • %s\n", v.String())
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Validated %d plan document(s) against repository structure rules:\n", len(checkedFiles))
	for _, f := range checkedFiles {
		fmt.Printf("  • %s\n", f)
	}
}

func isArtifactPath(p string) bool {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	return strings.Contains(abs, "/.gemini/antigravity/brain/")
}
