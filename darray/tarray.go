package darray

import (
	"context"
	"errors"
	"reflect"

	"github.com/iamlongalong/diskv/gkv"
)

func NewTArray[T any](_ T, darr *DArray) *TArray[T] {
	return &TArray[T]{d: darr}
}

func NewT[T any](darr *DArray) *TArray[T] {
	return &TArray[T]{d: darr}
}

type TArray[T any] struct {
	d *DArray
}

func (ta *TArray[T]) Get(ctx context.Context, index int) (*T, error) {
	data, err := ta.d.Get(ctx, index)
	if err != nil {
		return new(T), err
	}

	var t T

	// 2. 先看自身是否实现了 TMarshaler 接口
	if marshaler, ok := any(t).(gkv.TMarshaler); ok {
		err = marshaler.Unmarshal(data)
		if err != nil {
			return &t, err
		}

		return &t, nil
	}

	// 3. 看是否注册了全局的 Marshaler
	if unMarshaler, ok := gkv.GetUnMarshaler(t); ok {
		err := unMarshaler(data, reflect.ValueOf(t))
		if err != nil {
			return &t, err
		}

		return &t, nil
	}

	return &t, errors.New("not found marshaler")
}

func (ta *TArray[T]) Set(ctx context.Context, index int, v *T) error {
	if v == nil {
		// equal as del
		return ta.d.Set(ctx, index, nil)
	}

	// 2. 先看自身是否实现了 TMarshaler 接口
	if marshaler, ok := any(v).(gkv.TMarshaler); ok {
		data, err := marshaler.Marshal()
		if err != nil {
			return err
		}

		return ta.d.Set(ctx, index, data)
	}

	// 3. 看是否注册了全局的 GMarshaler
	if marshaler, ok := gkv.GetMarshaler(v); ok {
		data, err := marshaler(reflect.ValueOf(v))
		if err != nil {
			return err
		}

		return ta.d.Set(ctx, index, data)
	}

	return errors.New("not found marshaler")
}
