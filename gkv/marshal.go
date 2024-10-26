package gkv

import (
	"encoding/json"
	"reflect"
)

// normal marshaler
// 注意: 接收的 v any 都将是值的指针
type NMarshaler interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// reflect marshaler
type rMarshaler interface {
	Marshal(v reflect.Value) (data []byte, err error)
	Unmarshal([]byte, reflect.Value) error
}

type rMarshalerFunc = func(v reflect.Value) (data []byte, err error)
type rUnmarshalerFunc = func(data []byte, v reflect.Value) (err error)

// generic marshaler
type GMarshaler[T any] interface {
	Marshal(v *T) ([]byte, error)
	Unmarshal(data []byte, v *T) (err error)
}

// gMarshaler[T]
// gUnmarshaler[T]

type gMarshaler[T any] interface {
	Marshal(v *T) ([]byte, error)
}

type gUnmarshaler[T any] interface {
	Unmarshal(data []byte, v *T) (err error)
}

// type marshaler
// 实现了自身的 marshaler
// update: 暂时不支持，可用 GMarshaler 实现
type TMarshaler interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

func init() {
	dfRegistry.setDefault(dfJSONMarshaler)
}

func SetDefaultMarshaler(marshaler NMarshaler) {
	dfRegistry.setDefault(marshaler)
}
func RegisterMarshaler(t any, marshaler NMarshaler) {
	dfRegistry.registerMarshalerFunc(t,
		func(v reflect.Value) (data []byte, err error) {
			return marshaler.Marshal(v.Interface())
		},
		func(data []byte, v reflect.Value) (err error) {
			err = marshaler.Unmarshal(data, v.Interface())
			if err != nil {
				return err
			}

			return nil
		},
	)
}

func RegisterGMarshaler[T any](marshaler GMarshaler[T]) {
	t := *new(T)
	// fmt.Printf("register marshaler for type: %v\n", reflect.TypeOf(t).String())
	dfRegistry.registerMarshalerFunc(t,
		func(v reflect.Value) (data []byte, err error) {
			t := v.Interface().(*T)
			return marshaler.Marshal(t)
		},
		func(data []byte, v reflect.Value) (err error) {
			t := v.Interface().(*T)
			err = marshaler.Unmarshal(data, t)
			if err != nil {
				return err
			}

			return nil
		})
}

func GetUnMarshaler(t any) (rUnmarshalerFunc, bool) {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	fn := dfRegistry.getUnMarshaler(typ)
	return fn, fn != nil
}

func GetMarshaler(t any) (rMarshalerFunc, bool) {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	fn := dfRegistry.getMarshaler(typ)
	return fn, fn != nil
}

type marshalerRegistry struct {
	dfMarshal   rMarshalerFunc
	dfUnmarshal rUnmarshalerFunc

	marshalers   map[reflect.Type]rMarshalerFunc
	unmarshalers map[reflect.Type]rUnmarshalerFunc
}

var dfRegistry = marshalerRegistry{
	marshalers:   make(map[reflect.Type]rMarshalerFunc),
	unmarshalers: make(map[reflect.Type]rUnmarshalerFunc),
}

func (mr *marshalerRegistry) registerMarshaler(t any, marshaler rMarshaler) {
	typ := reflect.TypeOf(t)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// fmt.Printf("register marshaler for type: %v\n", typ.String())
	mr.marshalers[typ] = marshaler.Marshal
	mr.unmarshalers[typ] = marshaler.Unmarshal
}

func (mr *marshalerRegistry) registerMarshalerFunc(t any, marshaler rMarshalerFunc, unmarshaler rUnmarshalerFunc) {
	typ := reflect.TypeOf(t)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	// fmt.Printf("register marshaler for type: %v\n", typ.String())
	mr.marshalers[typ] = marshaler
	mr.unmarshalers[typ] = unmarshaler
}

func (mr *marshalerRegistry) setDefault(marshaler NMarshaler) {
	dfRegistry.dfMarshal = func(v reflect.Value) (data []byte, err error) {
		return marshaler.Marshal(v.Interface())
	}
	dfRegistry.dfUnmarshal = func(data []byte, v reflect.Value) (err error) {
		return marshaler.Unmarshal(data, v.Interface())
	}
}

func (mr *marshalerRegistry) getMarshaler(typ reflect.Type) rMarshalerFunc {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	// fmt.Printf("get marshaler for type: %v\n", typ.String())

	if fn, ok := mr.marshalers[typ]; ok {
		return fn
	}

	return mr.dfMarshal
}

func (mr *marshalerRegistry) getUnMarshaler(typ reflect.Type) rUnmarshalerFunc {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// fmt.Printf("get unmarshaler for type: %v\n", typ.String())
	if fn, ok := mr.unmarshalers[typ]; ok {
		return fn
	}

	return mr.dfUnmarshal
}

// default json marshaler

var dfJSONMarshaler = &jsonMarshaler{}

type jsonMarshaler struct{}

func (jm jsonMarshaler) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (jm jsonMarshaler) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
