package primitives_test

import (
	"testing"

	"last-mile-task/internal/primitives"
)

func TestSnowflakeGen_NextMonotonic(t *testing.T) {
	gen := primitives.NewSnowflakeGen(1)

	prev := gen.Next()
	for i := 0; i < 50; i++ {
		next := gen.Next()
		if next <= prev {
			t.Fatalf("Next() is not monotonic: %d then %d", prev, next)
		}
		prev = next
	}
}

func TestSnowflakeGen_WorkerIDBounds(t *testing.T) {
	for _, workerID := range []int64{0, 15} {
		// Valid worker ids must not panic.
		primitives.NewSnowflakeGen(workerID)
	}

	for _, workerID := range []int64{-1, 16, 100} {
		workerID := workerID
		t.Run("invalid worker", func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("NewSnowflakeGen(%d) did not panic", workerID)
				}
			}()
			primitives.NewSnowflakeGen(workerID)
		})
	}
}

func TestNextSnowflake_Monotonic(t *testing.T) {
	prev := primitives.NextSnowflake()
	for i := 0; i < 20; i++ {
		next := primitives.NextSnowflake()
		if next <= prev {
			t.Fatalf("NextSnowflake() is not monotonic: %d then %d", prev, next)
		}
		prev = next
	}
}