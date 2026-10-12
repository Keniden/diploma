package tsd

import "last-mile-task/internal/shipping/domain/values"

type Package struct {
	ID        values.PackageID
	TareID    values.TareID
	IsScanned bool
}
