// Deprecated wrapper: use event-pipeline/internal/consumer instead.
// This file re-exports the internal package to avoid duplication and keep APIs compatible.
package consumer

import (
	iconsumer "event-pipeline/internal/consumer"
)

// Type alias to the canonical internal implementation
type Consumer = iconsumer.Consumer

// Constructor alias
var New = iconsumer.New

// Methods are preserved via type alias.
