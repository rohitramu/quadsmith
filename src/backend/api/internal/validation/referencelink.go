package validation

import (
	"errors"
	"fmt"
	"strings"

	pb "quadsmith/api/gen/quadsmith"
)

// ErrUnspecifiedReferenceLinkType is returned when a reference link explicitly contains REFERENCE_LINK_TYPE_UNSPECIFIED.
var ErrUnspecifiedReferenceLinkType = errors.New("REFERENCE_LINK_TYPE_UNSPECIFIED is not permitted; omit categories or provide an empty types array for 'Other'")

// ErrDuplicateReferenceLinkURL is returned when multiple reference links have the same URL.
var ErrDuplicateReferenceLinkURL = errors.New("duplicate reference link URL is not permitted; combine categories into the types array of a single link")

// ValidateReferenceLinks checks that:
// 1. None of the provided reference links contain REFERENCE_LINK_TYPE_UNSPECIFIED.
// 2. No duplicate URLs are provided within the same set of reference links.
func ValidateReferenceLinks(links []*pb.ReferenceLink) error {
	seenURLs := make(map[string]int)
	for i, link := range links {
		if link == nil {
			continue
		}
		for _, t := range link.Types {
			if t == pb.ReferenceLinkType_REFERENCE_LINK_TYPE_UNSPECIFIED {
				return fmt.Errorf("reference link %d: %w", i, ErrUnspecifiedReferenceLinkType)
			}
		}

		normURL := strings.TrimRight(strings.TrimSpace(link.Url), "/")
		if normURL != "" {
			if prevIdx, exists := seenURLs[normURL]; exists {
				return fmt.Errorf("reference link %d duplicates URL from link %d (%s): %w", i, prevIdx, link.Url, ErrDuplicateReferenceLinkURL)
			}
			seenURLs[normURL] = i
		}
	}
	return nil
}
