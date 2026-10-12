// Package qr handles QR-code handshake generation and parsing for the gate
// reserve flow (ARCHITECTURE.md §23 flows 4 and 7). A session token is a
// signed, URL-safe payload that the TSD scanner decodes to confirm a reserve.
//
// The payload is HMAC-SHA256 signed with the shared handshake secret so a
// tampered QR fails decryption instead of silently wielding a wrong session.
package qr

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ErrMalformedSession is returned when the QR token cannot be parsed or its
// signature does not match the shared secret.
var ErrMalformedSession = errors.New("malformed qr session")

// Session is the payload carried by a reserve QR-code handshake token.
type Session struct {
	ShipmentID   string    `json:"shipment_id"`
	RoutePointID string    `json:"route_point_id"`
	TaskID       string    `json:"task_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// CreateSession signs and encodes the payload into a QR token of the form
// <payload>.<signature>, both base64url-encoded.
func CreateSession(s Session, secret []byte) (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", err
	}

	body := base64.RawURLEncoding.EncodeToString(payload)
	sig := base64.RawURLEncoding.EncodeToString(sign(payload, secret))

	return body + "." + sig, nil
}

// DecodeQR verifies and decodes a QR token back into the payload.
func DecodeQR(token string, secret []byte) (Session, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Session{}, ErrMalformedSession
	}

	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Session{}, ErrMalformedSession
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Session{}, ErrMalformedSession
	}

	if !hmac.Equal(sig, sign(body, secret)) {
		return Session{}, ErrMalformedSession
	}

	var s Session
	if err := json.Unmarshal(body, &s); err != nil {
		return Session{}, ErrMalformedSession
	}

	return s, nil
}

func sign(data, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return mac.Sum(nil)
}