package validation

import (
	"errors"
	"testing"

	pb "quadsmith/api/gen/quadsmith"
)

func TestValidateReferenceLinks(t *testing.T) {
	tests := []struct {
		name    string
		links   []*pb.ReferenceLink
		wantErr bool
	}{
		{
			name:    "nil links slice is valid",
			links:   nil,
			wantErr: false,
		},
		{
			name:    "empty links slice is valid",
			links:   []*pb.ReferenceLink{},
			wantErr: false,
		},
		{
			name: "link with empty types array is valid ('Other')",
			links: []*pb.ReferenceLink{
				{
					Types: []pb.ReferenceLinkType{},
					Url:   "https://example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "link with valid types",
			links: []*pb.ReferenceLink{
				{
					Types: []pb.ReferenceLinkType{
						pb.ReferenceLinkType_REFERENCE_LINK_TYPE_PRODUCT_PAGE,
						pb.ReferenceLinkType_REFERENCE_LINK_TYPE_PURCHASE,
					},
					Url: "https://example.com/product",
				},
			},
			wantErr: false,
		},
		{
			name: "link containing UNSPECIFIED returns error",
			links: []*pb.ReferenceLink{
				{
					Types: []pb.ReferenceLinkType{
						pb.ReferenceLinkType_REFERENCE_LINK_TYPE_UNSPECIFIED,
					},
					Url: "https://example.com/invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "multi-type link containing UNSPECIFIED among valid types returns error",
			links: []*pb.ReferenceLink{
				{
					Types: []pb.ReferenceLinkType{
						pb.ReferenceLinkType_REFERENCE_LINK_TYPE_DOCUMENTATION,
						pb.ReferenceLinkType_REFERENCE_LINK_TYPE_UNSPECIFIED,
					},
					Url: "https://example.com/docs",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReferenceLinks(tt.links)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateReferenceLinks() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrUnspecifiedReferenceLinkType) {
				t.Errorf("expected ErrUnspecifiedReferenceLinkType, got: %v", err)
			}
		})
	}
}
