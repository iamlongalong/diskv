package glist

import (
	"context"
	"fmt"

	"github.com/iamlongalong/diskv/gkv"
	"github.com/iamlongalong/diskv/kvstore"
)

type GList[T any] struct {
	kv *gkv.Gkv[T]
}

func New[T any](store kvstore.KVStorer) *GList[T] {
	return &GList[T]{kv: gkv.New[T](store)}
}

func NewT[T any](t T, store kvstore.KVStorer) *GList[T] {
	return &GList[T]{kv: gkv.NewT(t, store)}
}

func (gl *GList[T]) Get(ctx context.Context, idx int) (*T, bool, error) {
	key := fmt.Sprintf("%d", idx)
	return gl.kv.Get(ctx, key)
}

func (gl *GList[T]) GetRange(ctx context.Context, start, end int) ([]*T, error) {
	return nil, nil
}

func (gl *GList[T]) Set(ctx context.Context, idx int, v *T) error {
	key := fmt.Sprintf("%d", idx)

	return gl.kv.Set(ctx, key, v)
}

func (gl *GList[T]) SetRange(ctx context.Context, start int, values []*T) error {
	return nil
}

func (gl *GList[T]) Append(ctx context.Context, v *T) (err error) {
	return nil
}

func (gl *GList[T]) Appends(ctx context.Context, values []*T) (err error) {
	return nil
}

func (gl *GList[T]) Prepend(ctx context.Context, v *T) (err error) {
	return nil
}

func (gl *GList[T]) Prepends(ctx context.Context, values []*T) (err error) {
	return nil
}

func (gl *GList[T]) Insert(ctx context.Context, idx int, v *T) error {
	return nil
}

func (gl *GList[T]) Inserts(ctx context.Context, idx int, values []*T) error {
	return nil
}

func (gl *GList[T]) Remove(ctx context.Context, idx int) error {
	return nil
}

func (gl *GList[T]) RemoveRange(ctx context.Context, start, end int) error {
	return nil
}

func (gl *GList[T]) RemoveAll(ctx context.Context) error {
	return nil
}

func (gl *GList[T]) Len(ctx context.Context) (int, error) {
	return 0, nil
}

func (gl *GList[T]) Range(ctx context.Context, start, end int, fn func(ctx context.Context, idx int, v *T) bool) error {
	return nil
}

func (gl *GList[T]) RangeAll(ctx context.Context, fn func(ctx context.Context, idx int, v *T) bool) error {
	return nil
}
