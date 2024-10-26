package gkv

import (
	"context"
	"errors"
	"reflect"

	"github.com/iamlongalong/diskv/kvstore"
)

type Gkv[T any] struct {
	store kvstore.KVStorer
}

func New[T any](store kvstore.KVStorer) *Gkv[T] {
	return &Gkv[T]{store: store}
}

func NewT[T any](_ T, store kvstore.KVStorer) *Gkv[T] {
	return &Gkv[T]{store: store}
}

func (gd *Gkv[T]) Get(ctx context.Context, key string) (*T, bool, error) {
	t := new(T)

	data, ok, err := gd.store.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	// 2. 先看自身是否实现了 TMarshaler 接口
	if marshaler, ok := any(t).(TMarshaler); ok {
		err = marshaler.Unmarshal(data)
		if err != nil {
			return nil, false, err
		}

		return t, true, nil
	}

	// 3. 看是否注册了全局的 Marshaler
	if unMarshaler, ok := GetUnMarshaler(t); ok {
		err := unMarshaler(data, reflect.ValueOf(t))
		if err != nil {
			return nil, false, err
		}

		return t, true, nil
	}

	return t, false, errors.New("not found marshaler")
}

func (gd *Gkv[T]) Set(ctx context.Context, key string, v *T) error {
	if v == nil {
		// equal as del
		_, err := gd.store.Del(ctx, key)
		return err
	}

	// 2. 先看自身是否实现了 TMarshaler 接口
	if marshaler, ok := any(v).(TMarshaler); ok {
		data, err := marshaler.Marshal()
		if err != nil {
			return err
		}

		return gd.store.Set(ctx, key, data)
	}

	// 3. 看是否注册了全局的 GMarshaler
	if marshaler, ok := GetMarshaler(v); ok {
		data, err := marshaler(reflect.ValueOf(v))
		if err != nil {
			return err
		}

		return gd.store.Set(ctx, key, data)
	}

	return errors.New("not found marshaler")
}

// Nkv, 无须在初始化时指定类型，根据传入的 v 的类型匹配
type Nkv struct {
	store kvstore.KVStorer
}

func NewNkv(store kvstore.KVStorer) *Nkv {
	return &Nkv{store: store}
}

func (nd *Nkv) Get(ctx context.Context, key string, v any) (bool, error) {
	// 检查 v 是否是指针
	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Ptr {
		return false, errors.New("v must be pointer")
	}

	data, ok, err := nd.store.Get(ctx, key)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	// 1. 看自身是否实现了 TMarshaler 接口
	if marshaler, ok := any(v).(TMarshaler); ok {
		err := marshaler.Unmarshal(data)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	// 2. 看是否注册了全局的 Marshaler
	fn := dfRegistry.getUnMarshaler(typ)
	if fn == nil {
		return false, errors.New("not found marshaler")
	}

	return true, fn(data, reflect.ValueOf(v))
}

func (nd *Nkv) Set(ctx context.Context, key string, v any) error {
	if v == nil {
		_, err := nd.store.Del(ctx, key)
		return err
	}

	val := reflect.ValueOf(v)

	// 检查是否为指针，若不是则获取其指针
	if val.Kind() != reflect.Ptr {
		ptrVal := reflect.New(val.Type())
		ptrVal.Elem().Set(val)
		val = ptrVal
	}

	// 1. 看自身是否实现了 TMarshaler 接口
	if marshaler, ok := any(v).(TMarshaler); ok {
		data, err := marshaler.Marshal()
		if err != nil {
			return err
		}

		return nd.store.Set(ctx, key, data)
	}

	// 2. 看是否注册了全局的 Marshaler
	if fn, ok := GetMarshaler(v); ok {
		data, err := fn(val)
		if err != nil {
			return err
		}

		return nd.store.Set(ctx, key, data)
	}

	return errors.New("not found marshaler")

}
