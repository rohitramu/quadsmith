package pagination

import (
	"encoding/base64"
	"testing"
)

func TestParsePageSize(t *testing.T) {
	// Negative page size
	if _, err := ParsePageSize(-1); err == nil {
		t.Fatalf("expected error for negative page size, got nil")
	}

	// Zero defaults to DefaultPageSize
	size, err := ParsePageSize(0)
	if err != nil {
		t.Fatalf("unexpected error for 0 page size: %v", err)
	}
	if size != DefaultPageSize {
		t.Fatalf("expected default page size %d, got %d", DefaultPageSize, size)
	}

	// Normal valid page size
	size, err = ParsePageSize(25)
	if err != nil {
		t.Fatalf("unexpected error for valid page size: %v", err)
	}
	if size != 25 {
		t.Fatalf("expected page size 25, got %d", size)
	}

	// Capped to MaxPageSize
	size, err = ParsePageSize(200)
	if err != nil {
		t.Fatalf("unexpected error for large page size: %v", err)
	}
	if size != MaxPageSize {
		t.Fatalf("expected max page size %d, got %d", MaxPageSize, size)
	}
}

func TestEncodeDecodePageToken(t *testing.T) {
	filter := `manufacturer == "T-Motor"`
	sort := []string{"^kv", "id"}
	offset := int32(40)

	tokenStr, err := EncodePageToken(offset, filter, sort)
	if err != nil {
		t.Fatalf("failed to encode token: %v", err)
	}

	decoded, err := DecodePageToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to decode token: %v", err)
	}

	if decoded.Offset != offset {
		t.Errorf("expected offset %d, got %d", offset, decoded.Offset)
	}
	if decoded.Filter != filter {
		t.Errorf("expected filter %q, got %q", filter, decoded.Filter)
	}
	if len(decoded.Sort) != len(sort) || decoded.Sort[0] != sort[0] || decoded.Sort[1] != sort[1] {
		t.Errorf("expected sort %v, got %v", sort, decoded.Sort)
	}
}

func TestDecodePageToken_Invalid(t *testing.T) {
	// Empty token string
	decoded, err := DecodePageToken("")
	if err != nil || decoded != nil {
		t.Fatalf("expected nil for empty token string, got %v, err=%v", decoded, err)
	}

	// Not base64
	if _, err := DecodePageToken("!not_base64!"); err == nil {
		t.Fatalf("expected error for non-base64 string")
	}

	// Not JSON
	invalidJSON := base64.RawURLEncoding.EncodeToString([]byte("hello world"))
	if _, err := DecodePageToken(invalidJSON); err == nil {
		t.Fatalf("expected error for non-JSON string")
	}

	// Negative offset
	negativeOffsetJSON := base64.RawURLEncoding.EncodeToString([]byte(`{"offset":-5}`))
	if _, err := DecodePageToken(negativeOffsetJSON); err == nil {
		t.Fatalf("expected error for negative offset")
	}
}

func TestResolveParams(t *testing.T) {
	filter := `weight_g < 35.0`
	sort := []string{"^kv"}

	tokenStr, err := EncodePageToken(20, filter, sort)
	if err != nil {
		t.Fatalf("failed to encode token: %v", err)
	}

	// 1. First page: no token
	f, s, off, err := ResolveParams(filter, sort, "")
	if err != nil {
		t.Fatalf("unexpected error for empty token: %v", err)
	}
	if f != filter || off != 0 || len(s) != 1 || s[0] != "^kv" {
		t.Fatalf("unexpected resolved params for empty token: f=%q, off=%d, s=%v", f, off, s)
	}

	// 2. Matching request filter and sort
	f, s, off, err = ResolveParams(filter, sort, tokenStr)
	if err != nil {
		t.Fatalf("unexpected error for matching token: %v", err)
	}
	if f != filter || off != 20 || len(s) != 1 || s[0] != "^kv" {
		t.Fatalf("unexpected resolved params: f=%q, off=%d, s=%v", f, off, s)
	}

	// 3. Token carries filter and sort when client leaves request filter/sort empty
	f, s, off, err = ResolveParams("", nil, tokenStr)
	if err != nil {
		t.Fatalf("unexpected error when request omits filter/sort: %v", err)
	}
	if f != filter || off != 20 || len(s) != 1 || s[0] != "^kv" {
		t.Fatalf("expected adopted filter and sort: f=%q, off=%d, s=%v", f, off, s)
	}

	// 4. Mismatched filter returns error
	_, _, _, err = ResolveParams(`weight_g < 20.0`, sort, tokenStr)
	if err == nil {
		t.Fatalf("expected error for mismatched filter, got nil")
	}

	// 5. Mismatched sort returns error
	_, _, _, err = ResolveParams(filter, []string{"weight_g"}, tokenStr)
	if err == nil {
		t.Fatalf("expected error for mismatched sort, got nil")
	}
}
