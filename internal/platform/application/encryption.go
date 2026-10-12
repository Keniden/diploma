package application

// createEncryptionService builds the QR/claims encryption service used by the
// driver handshake.
//
// The real repository wires pkg/encryption.Service with the config secrets;
// pkg/encryption is not ported into the skeleton yet. TODO(LASTMILE-098):
// return the real service once pkg/encryption and config.Encryption land.
//
//nolint:unused // TODO(LASTMILE-098): wired once the dependent services land.
func (a *Application) createEncryptionService() (any, error) {
	// TODO(LASTMILE-098)
	return nil, nil
}
