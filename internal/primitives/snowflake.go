package primitives

import (
	"sync"
	"time"
)

const (
	snowflakeEpoch = 1780272000 // 2026-06-01T00:00:00Z
	workerBits     = 4
	seqBits        = 4
	maxWorkerID    = (1 << workerBits) - 1
	maxSeq         = (1 << seqBits) - 1
	snowflakeShift = workerBits + seqBits
)

// SnowflakeGen is a snowflake identifier generator (epoch-based, worker and
// sequence bits). It is safe for concurrent use.
type SnowflakeGen struct {
	mu       sync.Mutex
	epoch    int64
	workerID int64
	seq      int64
	lastTs   int64
}

// NewSnowflakeGen builds a generator for the given worker id (0..15).
func NewSnowflakeGen(workerID int64) *SnowflakeGen {
	if workerID < 0 || workerID > maxWorkerID {
		panic("snowflake workerID must be in [0, 15]")
	}
	return &SnowflakeGen{
		epoch:    snowflakeEpoch,
		workerID: workerID << snowflakeShift,
	}
}

// Next returns the next monotonically increasing snowflake id.
//
// TODO(LASTMILE-016): mirror the real generator exactly (component layout,
// overflow handling) once the reference implementation is verified.
func (g *SnowflakeGen) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().Unix()
	if now < g.lastTs {
		now = g.lastTs // clock moved backwards; keep monotonicity
	}
	if now == g.lastTs {
		g.seq = (g.seq + 1) & maxSeq
		if g.seq == 0 {
			for now <= g.lastTs {
				now = time.Now().Unix()
			}
		}
	} else {
		g.seq = 0
	}
	g.lastTs = now

	return now<<32 | g.workerID | g.seq
}

var defaultGen = NewSnowflakeGen(0)

// NextSnowflake returns the next id from the process-wide generator.
func NextSnowflake() int64 {
	return defaultGen.Next()
}