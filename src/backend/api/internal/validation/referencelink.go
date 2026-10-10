package validation

import (
	"errors"
	"fmt"
	pb "quadsmith/api/gen/quadsmith"
)

// ErrUnspecifiedReferenceLinkType is returned when a reference link explicitly contains REFERENCE_LINK_TYPE_UNSPECIFIED.
var ErrUnspecifiedReferenceLinkType = errors.New("REFERENCE_LINK_TYPE_UNSPECIFIED is not permitted; omit categories or provide an empty types array for 'Other'")

// ValidateReferenceLinks checks that none of the provided reference links contain REFERENCE_LINK_TYPE_UNSPECIFIED.
func ValidateReferenceLinks(links []*pb.ReferenceLink) error {
	for i, link := range links {
		if link == nil {
			continue
		}
		for _, t := range link.Types {
			if t == pb.ReferenceLinkType_REFERENCE_LINK_TYPE_UNSPECIFIED {
				return fmt.Errorf("reference link %d: %w", i, ErrUnspecifiedReferenceLinkType)
			}
		}
	}
	return nil
}
