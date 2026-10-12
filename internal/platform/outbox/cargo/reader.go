package cargo

import (
	"context"
	"encoding/json"

	"last-mile-task/pkg/outbox"
)

// CargoClient is the cargo-core API surface the readers need. The real
// repository binds cargo_core_client_v1.Client.UpdatePackagesStatusByAction
// here; the skeleton's cargo-core client is not ported yet, so a narrow
// interface is used until it lands. TODO(LASTMILE-054): switch to
// cargo_core_client_v1.Client.
type CargoClient interface {
	UpdatePackagesStatusByAction(ctx context.Context, packages []UpdatePackagesStatusPackage) error
}

func NewScannedPackagesReader(
	querier outbox.Querier,
	client CargoClient,
) *outbox.Reader {
	return outbox.NewReader(
		outbox.NewPostgresOutbox(querier, "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, message *outbox.Message) error {
			var msg ScannedPackagesMessage

			if err := json.Unmarshal(message.Payload, &msg); err != nil {
				return err
			}

			return client.UpdatePackagesStatusByAction(ctx, msg.Packages)
		}),
		outbox.ReaderWithMessageType(MessageTypeScannedPackages),
	)
}

func NewPackagesStatusReader(
	querier outbox.Querier,
	client CargoClient,
) *outbox.Reader {
	return outbox.NewReader(
		outbox.NewPostgresOutbox(querier, "lmt.outbox"),
		outbox.MessageHandlerFunc(func(ctx context.Context, message *outbox.Message) error {
			var msg PackagesStatusMessage

			if err := json.Unmarshal(message.Payload, &msg); err != nil {
				return err
			}

			return client.UpdatePackagesStatusByAction(ctx, msg.Packages)
		}),
		outbox.ReaderWithMessageType(MessageTypePackagesStatusUpdated),
	)
}
