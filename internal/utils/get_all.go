package utils

import (
	"last-mile-task/internal/primitives"
)

// GetAll walks cursor-based pagination pages until the last page is reached.
//
// TODO(LASTMILE-070): adapted to the current internal/primitives contract
// (Page.Token, PaginationResponse.HasMore/NextPage). The reference
// implementation uses Page.PageToken / PaginationResponse.NextPageToken;
// unify once internal/primitives converges.
func GetAll[T1 any, T2 comparable](
	f func(primitives.Page[T2]) ([]T1, primitives.PaginationResponse[T2], error),
	page primitives.Page[T2],
) ([]T1, error) {
	slice := make([]T1, 0)
	hasNext := true
	for hasNext {
		values, pageRes, err := f(page)
		if err != nil {
			return nil, err
		}
		slice = append(slice, values...)
		hasNext = pageRes.HasMore
		if pageRes.NextPage != nil {
			page.Token = pageRes.NextPage.Token
		}
		if len(values) == 0 {
			return slice, nil
		}
	}

	return slice, nil
}
