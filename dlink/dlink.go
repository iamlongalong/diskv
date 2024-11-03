package dlink

import "github.com/iamlongalong/diskv/kvstore"

// for linked list
type DLink[T any] struct {
	store kvstore.KVStorer
}

func NewDLink[T any](store kvstore.KVStorer) *DLink[T] {
	return &DLink[T]{
		store: store,
	}
}

func (d *DLink[T]) Get(key string) (T, bool) {
	var t T
	return t, false
}

func (d *DLink[T]) Set(key string, value T) bool {
	return true
}

func (d *DLink[T]) Delete(key string) bool {
	return true
}
