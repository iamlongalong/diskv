package kvstore

import "context"

// KVStorer is a simple key-value store interface.
// It is used by the gdiskv package to store the values.
// diskv implements this interface.
// more implementations can be `redis`,`sqlite3`,`ectd`,`bbolt` and other kv stores.
type KVStorer interface {
	Has(ctx context.Context, key string) (has bool, err error)
	Get(ctx context.Context, key string) (data []byte, ok bool, err error)
	Set(ctx context.Context, key string, val []byte) error
	Del(ctx context.Context, key string) (ok bool, err error)
	ForEach(ctx context.Context, fn func(ctx context.Context, key string, value []byte) (ok bool)) error
}
