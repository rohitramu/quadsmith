package schemacheck

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	datapb "quadsmith/shared/gen/quadsmith/data"
	sqlpb "quadsmith/shared/gen/quadsmith/sql"
)

// VerifiedTableData records a verified SQL table message and its data classification.
type VerifiedTableData struct {
	MessageName string
	TableName   string
	Domain      datapb.ExportDomain
	FieldCount  int
}

// CheckAllProtoDataClassification validates that all SQL-backed Protobuf messages and their fields
// conform strictly to GDPR classification invariants declared in quadsmith.data:
// 1. Every (quadsmith.sql.table) message must declare a non-unspecified (quadsmith.data.export).domain.
// 2. Every field on a table message (and referenced sub-message) must have an explicit content_type.
// 3. ID and relational key provenance rules are enforced (uuid as SYSTEM_METADATA_PUBLIC, etc.).
// 4. Operations schema isolation: quadsmith.ops tables must declare EXPORT_DOMAIN_NONE with 0 USER_* fields.
func CheckAllProtoDataClassification() ([]VerifiedTableData, error) {
	var verified []VerifiedTableData

	checkedSubMessages := make(map[string]bool)

	var checkSubMessageFields func(md protoreflect.MessageDescriptor) error
	checkSubMessageFields = func(md protoreflect.MessageDescriptor) error {
		msgName := string(md.FullName())
		if checkedSubMessages[msgName] {
			return nil
		}
		checkedSubMessages[msgName] = true

		// Skip google.protobuf types
		if strings.HasPrefix(msgName, "google.protobuf.") {
			return nil
		}

		fields := md.Fields()
		for i := 0; i < fields.Len(); i++ {
			field := fields.Get(i)
			opts := field.Options()
			if !proto.HasExtension(opts, datapb.E_ContentType) {
				return fmt.Errorf("embedded field %s.%s missing (quadsmith.data.content_type)", msgName, field.Name())
			}
			ct := proto.GetExtension(opts, datapb.E_ContentType).(datapb.ContentType)
			if ct == datapb.ContentType_CONTENT_TYPE_UNSPECIFIED {
				return fmt.Errorf("embedded field %s.%s has CONTENT_TYPE_UNSPECIFIED", msgName, field.Name())
			}
			if field.Kind() == protoreflect.MessageKind && field.Message() != nil {
				if err := checkSubMessageFields(field.Message()); err != nil {
					return err
				}
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
			md := msgs.Get(i)
			opts := md.Options()

			if !proto.HasExtension(opts, sqlpb.E_Table) {
				continue
			}
			tblOpt := proto.GetExtension(opts, sqlpb.E_Table).(*sqlpb.TableOptions)
			if tblOpt == nil || tblOpt.Name == "" {
				continue
			}

			// Rule 1: Table domain coverage
			if !proto.HasExtension(opts, datapb.E_Export) {
				firstErr = fmt.Errorf("table message %s (table %s) missing option (quadsmith.data.export)", md.FullName(), tblOpt.Name)
				return false
			}
			exportOpt := proto.GetExtension(opts, datapb.E_Export).(*datapb.ExportOptions)
			if exportOpt == nil || exportOpt.Domain == datapb.ExportDomain_EXPORT_DOMAIN_UNSPECIFIED {
				firstErr = fmt.Errorf("table message %s (table %s) has EXPORT_DOMAIN_UNSPECIFIED", md.FullName(), tblOpt.Name)
				return false
			}

			isOps := strings.HasPrefix(string(md.FullName()), "quadsmith.ops.") || strings.HasSuffix(string(fd.Path()), "store/ops.proto")
			if isOps && exportOpt.Domain != datapb.ExportDomain_EXPORT_DOMAIN_NONE {
				firstErr = fmt.Errorf("ops table message %s must declare EXPORT_DOMAIN_NONE, got %v", md.FullName(), exportOpt.Domain)
				return false
			}

			fields := md.Fields()
			fieldCount := fields.Len()
			for j := 0; j < fieldCount; j++ {
				f := fields.Get(j)
				fOpts := f.Options()

				// Rule 2: Explicit field classification
				if !proto.HasExtension(fOpts, datapb.E_ContentType) {
					firstErr = fmt.Errorf("table field %s.%s missing (quadsmith.data.content_type)", md.FullName(), f.Name())
					return false
				}
				ct := proto.GetExtension(fOpts, datapb.E_ContentType).(datapb.ContentType)
				if ct == datapb.ContentType_CONTENT_TYPE_UNSPECIFIED {
					firstErr = fmt.Errorf("table field %s.%s has CONTENT_TYPE_UNSPECIFIED", md.FullName(), f.Name())
					return false
				}

				// Check embedded sub-messages
				if f.Kind() == protoreflect.MessageKind && f.Message() != nil {
					if err := checkSubMessageFields(f.Message()); err != nil {
						firstErr = err
						return false
					}
				}

				// Rule 5: Ops schema isolation (no USER_* fields)
				if isOps {
					if ct == datapb.ContentType_CONTENT_TYPE_USER_PROVIDED_PRIVATE ||
						ct == datapb.ContentType_CONTENT_TYPE_USER_PROVIDED_PUBLIC ||
						ct == datapb.ContentType_CONTENT_TYPE_USER_OBSERVED {
						firstErr = fmt.Errorf("ops table field %s.%s cannot be a USER_* content type: %v", md.FullName(), f.Name(), ct)
						return false
					}
				}

				// Rule 3: Identifier & Relational Key Provenance checks
				fName := string(f.Name())
				if fName == "uuid" && ct != datapb.ContentType_CONTENT_TYPE_SYSTEM_METADATA_PUBLIC {
					firstErr = fmt.Errorf("primary key field %s.%s must be SYSTEM_METADATA_PUBLIC, got %v", md.FullName(), f.Name(), ct)
					return false
				}
				if fName == "id" {
					if md.FullName() == "quadsmith.store.User" {
						if ct != datapb.ContentType_CONTENT_TYPE_USER_PROVIDED_PUBLIC {
							firstErr = fmt.Errorf("User.id must be USER_PROVIDED_PUBLIC, got %v", ct)
							return false
						}
					} else if ct != datapb.ContentType_CONTENT_TYPE_SYSTEM_METADATA_PUBLIC {
						firstErr = fmt.Errorf("slug field %s.%s must be SYSTEM_METADATA_PUBLIC, got %v", md.FullName(), f.Name(), ct)
						return false
					}
				}
			}

			verified = append(verified, VerifiedTableData{
				MessageName: string(md.FullName()),
				TableName:   tblOpt.Name,
				Domain:      exportOpt.Domain,
				FieldCount:  fieldCount,
			})
		}
		return true
	})

	if firstErr != nil {
		return nil, firstErr
	}
	return verified, nil
}
