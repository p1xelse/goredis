package storage

import (
	"time"
)

type SetOptions struct {
	TTL        *time.Duration
	Expiration *int64
}
