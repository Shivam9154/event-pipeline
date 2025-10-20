// Deprecated wrapper: use event-pipeline/internal/producer instead.
// This file re-exports the internal package to avoid duplication and keep APIs compatible.
package producer

import (
	iproducer "event-pipeline/internal/producer"
)

// Type alias to the canonical internal implementation
type Producer = iproducer.Producer

// Constructor alias
var New = iproducer.New

// Method forwarding is not needed for type aliases; methods on iproducer.Producer are preserved.
