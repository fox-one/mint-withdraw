package mint

import (
	"context"
)

// Store store
type Store interface {
	ReadProperty(ctx context.Context, key string) (string, error)
	WriteProperty(ctx context.Context, key, value string) error
}
