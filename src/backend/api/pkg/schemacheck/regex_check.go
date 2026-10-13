package schemacheck

import (
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	validate "quadsmith/api/gen/quadsmith/validate"
)

// VerifiedRegex records a regex pattern found and verified in the Protobuf schema.
type VerifiedRegex struct {
	Field   string
	Pattern string
}

// CheckAllProtoRegexPatterns scans all Protobuf messages registered in protoregistry.GlobalFiles
// for (quadsmith.validate.rules).string.pattern options and validates that they successfully
// compile under Go's RE2 regex engine.
func CheckAllProtoRegexPatterns() ([]VerifiedRegex, error) {
	var verified []VerifiedRegex

	var checkMsg func(md protoreflect.MessageDescriptor) error
	checkMsg = func(md protoreflect.MessageDescriptor) error {
		fields := md.Fields()
		for i := 0; i < fields.Len(); i++ {
			field := fields.Get(i)
			opts := field.Options()
			if proto.HasExtension(opts, validate.E_Rules) {
				ext := proto.GetExtension(opts, validate.E_Rules)
				if rules, ok := ext.(*validate.FieldRules); ok && rules != nil && rules.GetString_() != nil {
					pattern := rules.GetString_().GetPattern()
					if pattern != "" {
						if _, err := regexp.Compile(pattern); err != nil {
							return fmt.Errorf("field %s.%s has invalid RE2 regex %q: %w", md.FullName(), field.Name(), pattern, err)
						}
						verified = append(verified, VerifiedRegex{
							Field:   fmt.Sprintf("%s.%s", md.FullName(), field.Name()),
							Pattern: pattern,
						})
					}
				}
			}
		}

		nested := md.Messages()
		for i := 0; i < nested.Len(); i++ {
			if err := checkMsg(nested.Get(i)); err != nil {
				return err
			}
		}
		return nil
	}

	var firstErr error
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if !strings.HasPrefix(string(fd.Package()), "quadsmith") {
			return true
		}
		msgs := fd.Messages()
		for i := 0; i < msgs.Len(); i++ {
			if err := checkMsg(msgs.Get(i)); err != nil {
				firstErr = err
				return false
			}
		}
		return true
	})

	if firstErr != nil {
		return nil, firstErr
	}
	return verified, nil
}
