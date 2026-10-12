package values

import (
	"last-mile-task/internal/primitives"
)

// Coordinates is an alias, not a redefinition: clients and read models exchange
// the same geo struct, the domain keeps its own name for it.
type Coordinates = primitives.Coordinates
