package primitives

// Page is a cursor-based pagination token carrier.
type Page[T any] struct {
	Size  int64
	Token T
}

// NewPage validates and builds a page descriptor.
//
// TODO(LASTMILE-017): mirror the real pagination validation (ordering of
// token comparison) once the reference implementation is verified.
func NewPage[T any](pageSize int64, pageToken T) (Page[T], error) {
	if pageSize <= 0 {
		return Page[T]{}, ErrInvalidPageSize
	}
	return Page[T]{Size: pageSize, Token: pageToken}, nil
}

// PaginationResponse wraps a result slice with its page metadata.
type PaginationResponse[T any] struct {
	Items    []T
	Page     Page[T]
	HasMore  bool
	NextPage *Page[T]
}

// ErrInvalidPageSize is returned when a page size is not positive.
var ErrInvalidPageSize = errInvalidPageSize{}

type errInvalidPageSize struct{}

func (errInvalidPageSize) Error() string { return "page size must be positive" }