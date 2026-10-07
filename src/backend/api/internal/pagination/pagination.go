package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const (
	// DefaultPageSize is the default number of items to return when page_size is 0 or unset.
	DefaultPageSize int32 = 20

	// MaxPageSize is the maximum allowed page size.
	MaxPageSize int32 = 100
)

// InvalidArgumentError represents an invalid request parameter error.
type InvalidArgumentError struct {
	msg string
}

func (e *InvalidArgumentError) Error() string {
	return e.msg
}

// ErrInvalidSortColumn creates an InvalidArgumentError for an unrecognized sort column.
func ErrInvalidSortColumn(col string) error {
	return &InvalidArgumentError{msg: fmt.Sprintf("invalid sort column %q", col)}
}

// IsInvalidArgument returns true if the error is an InvalidArgumentError.
func IsInvalidArgument(err error) bool {
	var invalidArg *InvalidArgumentError
	return errors.As(err, &invalidArg)
}

// PageToken represents the payload encoded inside an opaque page_token.
type PageToken struct {
	Offset int32    `json:"offset"`
	Filter string   `json:"filter,omitempty"`
	Sort   []string `json:"sort,omitempty"`
}

// ParsePageSize validates and returns the effective page size.
// Returns an error if pageSize < 0.
// Defaults to DefaultPageSize if pageSize == 0.
// Coerces to MaxPageSize if pageSize > MaxPageSize.
func ParsePageSize(pageSize int32) (int32, error) {
	if pageSize < 0 {
		return 0, errors.New("page_size cannot be negative")
	}
	if pageSize == 0 {
		return DefaultPageSize, nil
	}
	if pageSize > MaxPageSize {
		return MaxPageSize, nil
	}
	return pageSize, nil
}

// DecodePageToken decodes an opaque base64-encoded page token.
func DecodePageToken(tokenStr string) (*PageToken, error) {
	if tokenStr == "" {
		return nil, nil
	}
	var data []byte
	var err error
	data, err = base64.RawURLEncoding.DecodeString(tokenStr)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(tokenStr)
		if err != nil {
			data, err = base64.StdEncoding.DecodeString(tokenStr)
			if err != nil {
				return nil, fmt.Errorf("invalid page_token: %w", err)
			}
		}
	}

	var token PageToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("invalid page_token: %w", err)
	}
	if token.Offset < 0 {
		return nil, errors.New("invalid page_token: offset cannot be negative")
	}
	return &token, nil
}

// EncodePageToken serializes and base64-encodes a page token.
func EncodePageToken(offset int32, filter string, sort []string) (string, error) {
	token := PageToken{
		Offset: offset,
		Filter: filter,
		Sort:   sort,
	}
	data, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// ResolveParams resolves the effective filter, sort, and offset from the request and page_token.
// If page_token is provided, it validates that any explicitly passed filter and sort match the token.
func ResolveParams(reqFilter string, reqSort []string, reqTokenStr string) (filter string, sort []string, offset int32, err error) {
	if reqTokenStr == "" {
		return reqFilter, reqSort, 0, nil
	}

	token, err := DecodePageToken(reqTokenStr)
	if err != nil {
		return "", nil, 0, err
	}

	// Validate filter consistency: if request specifies a filter, it must match the token's filter
	if reqFilter != "" && reqFilter != token.Filter {
		return "", nil, 0, fmt.Errorf("filter in request (%q) does not match filter in page_token (%q)", reqFilter, token.Filter)
	}
	effectiveFilter := token.Filter
	if reqFilter != "" {
		effectiveFilter = reqFilter
	}

	// Validate sort consistency: if request specifies a sort, it must match the token's sort
	if len(reqSort) > 0 && !slices.Equal(reqSort, token.Sort) {
		return "", nil, 0, errors.New("sort in request does not match sort in page_token")
	}
	effectiveSort := token.Sort
	if len(reqSort) > 0 {
		effectiveSort = reqSort
	}

	return effectiveFilter, effectiveSort, token.Offset, nil
}
