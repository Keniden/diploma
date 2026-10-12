// Package primitives contains shared low-level abstractions: identifiers,
// snowflake generators, pagination, coordinates and time helpers.
package primitives

import "github.com/google/uuid"

// NewID returns a fresh UUID v7 identifier, falling back to a random UUID v4
// when the v7 generation is unavailable.
func NewID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}

	return id
}