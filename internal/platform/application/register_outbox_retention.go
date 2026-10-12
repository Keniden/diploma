package application

// registerOutboxRetention wires the outbox cleanup worker.
//
// The skeleton's platform/outbox/cleanup.Worker is a stub: the real worker
// needs config.OutboxCleanup (retention window + message types) and the
// batching delete over the repository. TODO(LASTMILE-096): enable the worker
// (via builder.StartBackground + OnShutdown) once cleanup.Run is implemented.
//
//nolint:unused // TODO(LASTMILE-096): wired once cleanup.Run is implemented.
func (a *Application) registerOutboxRetention() {
	// TODO(LASTMILE-096)
}
