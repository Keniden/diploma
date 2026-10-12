package bus

import (
	"last-mile-task/internal/shipping/infrastructure/events/bus/inmem"
)

var NewSyncEventBus = inmem.NewSyncEventBus
var NewAsyncEventBus = inmem.NewAsyncEventBus

type SyncEventBus = inmem.SyncEventBus
type AsyncEventBus = inmem.AsyncEventBus
type EventHandler = inmem.EventHandler
