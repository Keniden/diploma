package primitives_test

import (
	"testing"

	"github.com/google/uuid"

	"last-mile-task/internal/primitives"
)

func TestNewID_IsV7AndUnique(t *testing.T) {
	seen := make(map[uuid.UUID]struct{}, 100)
	for i := 0; i < 100; i++ {
		id := primitives.NewID()
		if id == uuid.Nil {
			t.Fatal("NewID() returned a nil UUID")
		}
		if id.Version() != 7 {
			t.Fatalf("NewID() version = %d, want 7", id.Version())
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("NewID() returned duplicate id %s", id)
		}
		seen[id] = struct{}{}
	}
}