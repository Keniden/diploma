package ports

import (
	"context"

	"last-mile-task/internal/shipping/domain/values"
)

type PackageProvider interface {
	ListReturnPackages(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ReturnPackage, error)
	ListForwardPackages(ctx context.Context, srcPointID values.PointID, dstPointIDs []values.PointID) ([]ForwardPackage, error)
	GetTareSticker(ctx context.Context, packageID values.PackageID) (string, error)
}

type ReturnPackage struct {
	ID       values.PackageID
	TareID   values.TareID
	TareType values.TareType
}

type ForwardPackage struct {
	ID         values.PackageID
	TareID     values.TareID
	DstPointID values.PointID
}
