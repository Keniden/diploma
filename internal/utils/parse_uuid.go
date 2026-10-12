package utils

import (
	"strings"

	"github.com/google/uuid"

	"last-mile-task/internal/errors"
)

// TODO(LASTMILE-070): errors.NewInvalidUuidError is not generated yet
// (cmd/errorgen is a stub and errors.yaml has no invalid-uuid entry).
// Replace with the generated error constructor once available.
func ParseUUID(u string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(u))
	if err != nil {
		return uuid.UUID{}, errors.New("failed to parse uuid", "invalid_uuid", 3 /* InvalidArgument */, 400).WithAttributes(
			errors.Attributes{
				"uuid": u,
			},
		)
	}

	return id, nil
}

func ParseUUIDList(raw []string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]uuid.UUID, 0, len(raw))
	for _, item := range raw {
		id, err := ParseUUID(item)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}
