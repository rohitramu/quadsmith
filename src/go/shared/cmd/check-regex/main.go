package main

import (
	"fmt"
	"os"

	"quadsmith/shared/pkg/schemacheck"

	_ "quadsmith/shared/gen/quadsmith/api"
)

func main() {
	verified, err := schemacheck.CheckAllProtoRegexPatterns()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Protobuf RE2 Regex Validation Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Verified %d Protobuf regex pattern(s) compile under Go RE2:\n", len(verified))
	for _, v := range verified {
		fmt.Printf("  • %s: %s\n", v.Field, v.Pattern)
	}
}
