package transport

import (
	"errors"
	"fmt"
)

// ErrNotImplemented is returned by transport providers that are registered
// by name but have no working implementation yet. It wraps
// errors.ErrUnsupported, so both errors.Is(err, ErrNotImplemented) and
// errors.Is(err, errors.ErrUnsupported) hold.
var ErrNotImplemented = fmt.Errorf("transport: provider not implemented: %w", errors.ErrUnsupported)
