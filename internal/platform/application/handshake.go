package application

import (
	"last-mile-task/pkg/qr"
)

// handshakeSecret is the slice-scoped stand-in for the handshake-validator
// shared secret configured via config.Config.Handshake in the real repository.
//
// TODO(LASTMILE-098): read the secret from config.Config.Handshake once the
// config block lands; the slice keeps a fixed test secret so the QR handshake
// is deterministic until then.
var handshakeSecret = []byte("last-mile-task-slice-handshake-secret")

// handshakeService signs and verifies reserve QR handshake sessions via pkg/qr.
type handshakeService struct {
	secret []byte
}

// CreateSession signs a reserve session into a QR token.
func (s *handshakeService) CreateSession(session qr.Session) (string, error) {
	return qr.CreateSession(session, s.secret)
}

// DecodeQR verifies a QR token and returns the embedded reserve session.
func (s *handshakeService) DecodeQR(token string) (qr.Session, error) {
	return qr.DecodeQR(token, s.secret)
}

// createHandshakeService builds the driver handshake service (QR code
// parsing in the real repository, pkg/qr).
func (a *Application) createHandshakeService() (any, error) {
	return &handshakeService{secret: handshakeSecret}, nil
}
