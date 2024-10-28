package gkv

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/iamlongalong/diskv/diskv"
)

type TStruct struct {
	A string            `json:"a"`
	B int               `json:"b"`
	C map[string]string `json:"c"`
}

// for gmarshaler test of TStruct
type TStructMarshal struct{}

func (tsm *TStructMarshal) Marshal(v *TStruct) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return b, nil
}
func (tsm *TStructMarshal) Unmarshal(data []byte, v *TStruct) (err error) {
	if v == nil {
		return errors.New("object for unmarshal is nil")
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	v.A += "_test"
	return nil
}

func NewTStructMarshal() GMarshaler[TStruct] {
	return &TStructMarshal{}
}

// for nmarshaler test of TNStruct
type TNStruct struct {
	A string            `json:"xa"`
	B int               `json:"xxb"`
	C map[string]string `json:"xxxc"`
}

type TNMarshaler struct{}

func (tm *TNMarshaler) Marshal(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	return b, err
}

func (tm *TNMarshaler) Unmarshal(data []byte, v any) error {
	tn, ok := v.(*TNStruct)
	if !ok {
		return errors.New("object type is not TNStruct")
	}

	err := json.Unmarshal(data, tn)
	if err != nil {
		return err
	}

	tn.A += "_intn"
	return nil
}

// test TMarshaler
type TSStruct struct {
	A string            `json:"a"`
	B int               `json:"b"`
	C map[string]string `json:"c"`
}

func (tsm *TSStruct) Marshal() ([]byte, error) {
	b, err := json.Marshal(tsm)
	return b, err
}

func (tsm *TSStruct) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, tsm)
	if err != nil {
		return err
	}

	tsm.A += "_self"
	return nil
}

func TestGDiskMarshaler(t *testing.T) {
	ctx := context.Background()
	disk, err := diskv.CreateDB(ctx, &diskv.CreateConfig{
		Dir:     "test",
		KeysLen: 100,
		MaxLen:  64,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test gdisk gmarshaler", func(t *testing.T) {
		RegisterGMarshaler(NewTStructMarshal())

		gd := New[TStruct](disk)
		t1 := &TStruct{
			A: "a",
			B: 1,
			C: map[string]string{
				"a": "a",
			},
		}

		err = gd.Set(ctx, "gdiskkey", t1)
		if err != nil {
			t.Fatal(err)
		}

		t2, ok, err := gd.Get(ctx, "gdiskkey")
		if err != nil {
			t.Fatal(err)
		}

		if !ok {
			t.Fatal("not found")
		}

		if t2.A != "a_test" {
			t.Fatal("unmarshal failed")
		}

		t.Log(t2)
	})

	t.Run("test gdisk nmarshaler", func(t *testing.T) {
		RegisterMarshaler(TNStruct{}, &TNMarshaler{})

		gd := New[TNStruct](disk)
		t1 := &TNStruct{
			A: "a",
			B: 1,
			C: map[string]string{
				"a": "a",
			},
		}

		err = gd.Set(ctx, "key_test_nmarshal", t1)
		if err != nil {
			t.Fatal(err)
		}

		t2, ok, err := gd.Get(ctx, "key_test_nmarshal")
		if err != nil {
			t.Fatal(err)
		}

		if !ok {
			t.Fatal("not found")
		}
		if t2.A != "a_intn" {
			t.Fatal("unmarshal failed")
		}
	})

	t.Run("test gdisk tmarshaler", func(t *testing.T) {
		gd := New[TSStruct](disk)
		t1 := &TSStruct{
			A: "a",
			B: 1,
			C: map[string]string{
				"a": "a",
			},
		}

		err = gd.Set(ctx, "key_test_tmarshal", t1)
		if err != nil {
			t.Fatal(err)
		}

		t2, ok, err := gd.Get(ctx, "key_test_tmarshal")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("not found")
		}

		if t2.A != "a_self" {
			t.Fatal("unmarshal failed")
		}

	})

	t.Run("test no val", func(t *testing.T) {
		gd := New[TSStruct](disk)
		v, ok, err := gd.Get(ctx, "key_test_tmarshal_no_val")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatal("should not found")
		}
		if v != nil {
			t.Fatal("should be nil")
		}
	})

	t.Run("test del val", func(t *testing.T) {
		gd := New[TSStruct](disk)
		err := gd.Set(ctx, "key_test_tmarshal_del_val", &TSStruct{A: "a_self"})
		if err != nil {
			t.Fatal(err)
		}

		_, ok, err := gd.Get(ctx, "key_test_tmarshal_del_val")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("should found")
		}

		err = gd.Set(ctx, "key_test_tmarshal_del_val", nil)
		if err != nil {
			t.Fatal(err)
		}

		_, ok, err = gd.Get(ctx, "key_test_tmarshal_del_val")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatal("should not found")
		}

	})

}

func TestNDisk(t *testing.T) {
	ctx := context.Background()
	db, err := diskv.CreateDB(ctx, &diskv.CreateConfig{
		Dir:     "test",
		KeysLen: 100,
		MaxLen:  64,
	})
	if err != nil {
		t.Fatal(err)
	}

	nd := NewNkv(db)

	t.Run("test_ndiskv", func(t *testing.T) {
		// tmarshaler
		RegisterMarshaler(TNStruct{}, &TNMarshaler{})
		tt := TNStruct{
			A: "a",
			B: 1,
			C: map[string]string{
				"a": "a",
			},
		}
		err = nd.Set(ctx, "key_test_tmarshal", tt)
		if err != nil {
			t.Fatal(err)
		}
		err = nd.Set(ctx, "key_test_tmarshal_ptr", &tt)
		if err != nil {
			t.Fatal(err)
		}
		tv := &TNStruct{}
		ok, err := nd.Get(ctx, "key_test_tmarshal", tv)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("not found")
		}
		if tv.A != "a_intn" {
			t.Fatal("unmarshal failed")
		}
	})

	t.Run("test_marshal_struct_ptr", func(t *testing.T) {
		RegisterGMarshaler(NewTStructMarshal())
		tg := &TStruct{
			A: "ga",
			B: 1,
			C: map[string]string{
				"a": "a",
			},
		}
		err = nd.Set(ctx, "key_test_tmarshal", tg)
		if err != nil {
			t.Fatal(err)
		}

		tsv := &TStruct{}
		ok, err := nd.Get(ctx, "key_test_tmarshal", tsv)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("not found")
		}

		if tsv.A != "ga_test" {
			t.Fatal("unmarshal failed")
		}

	})

	t.Run("test_marshaler_registry", func(t *testing.T) {
		// self marshal
		ts := &TSStruct{
			A: "tsa",
			B: 2,
			C: map[string]string{
				"b": "b",
			},
		}
		err = nd.Set(ctx, "key_test_smarshal", ts)
		if err != nil {
			t.Fatal(err)
		}
		tssv := &TSStruct{}
		ok, err := nd.Get(ctx, "key_test_smarshal", tssv)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("not found")
		}
		if tssv.A != "tsa_self" {
			t.Fatal("unmarshal failed")
		}

	})

	t.Run("test_default_marshal", func(t *testing.T) {
		// 使用 default marshaler (json marshaler)
		dm := map[string]string{
			"x":     "y",
			"hello": "world",
		}
		err = nd.Set(ctx, "key_test_default_marshaler", dm)
		if err != nil {
			t.Fatal(err)
		}
		dm2 := map[string]string{}
		ok, err := nd.Get(ctx, "key_test_default_marshaler", &dm2)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("not found")
		}
		if dm2["x"] != "y" || dm2["hello"] != "world" {
			t.Fatal("unmarshal failed")
		}
	})

	t.Run("test_mix_value_ptr", func(t *testing.T) {
		// 指针和值混用
		tsnp := TSStruct{A: "tsnp"}
		err = nd.Set(ctx, "key_test_default_marshaler_ptr", tsnp)
		if err != nil {
			t.Fatal(err)
		}
		tsnp2 := TSStruct{}
		ok, err := nd.Get(ctx, "key_test_default_marshaler_ptr", tsnp2)
		if err == nil { // should be error of non-pointer
			t.Fatal(err)
		}

		ok, err = nd.Get(ctx, "key_test_default_marshaler_ptr", &tsnp2)
		if err != nil {
			t.Fatal(err)
		}
		if !ok || tsnp2.A != "tsnp_self" {
			t.Fatal("not equal")
		}
	})

	t.Run("get nothing", func(t *testing.T) {
		ok, err := nd.Get(ctx, "key_not_exist", &TSStruct{})
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatal("should be false")
		}
	})

	t.Run("set base type value", func(t *testing.T) {
		err := nd.Set(ctx, "key_test_set_nil", nil)
		if err != nil {
			t.Fatal(err)
		}

		// set string
		err = nd.Set(ctx, "key_test_set_string", "test_set_string")
		if err != nil {
			t.Fatal(err)
		}

		var v string
		ok, err := nd.Get(ctx, "key_test_set_string", &v)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("should be true")
		}
		if v != "test_set_string" {
			t.Fatal("value should be test_set_string")
		}

		// set int
		err = nd.Set(ctx, "key_test_set_int", 123)
		if err != nil {
			t.Fatal(err)
		}
		var v2 int
		ok, err = nd.Get(ctx, "key_test_set_int", &v2)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("should be true")
		}
		if v2 != 123 {
			t.Fatal("value should be 123")
		}

		// set float64
		err = nd.Set(ctx, "key_test_set_float64", 123.456)
		if err != nil {
			t.Fatal(err)
		}
		var v3 float64
		ok, err = nd.Get(ctx, "key_test_set_float64", &v3)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("should be true")
		}
		if v3 != 123.456 {
			t.Fatal("value should be 123.456")
		}

	})

	t.Run("set nil for del", func(t *testing.T) {
		err := nd.Set(ctx, "key_test_del", 123)
		if err != nil {
			t.Fatal(err)
		}

		var i int
		ok, err := nd.Get(ctx, "key_test_del", &i)
		if err != nil {
			t.Fatal(err)
		}

		if !ok || i != 123 {
			t.Fatal("not equal")
		}

		err = nd.Set(ctx, "key_test_del", nil)
		if err != nil {
			t.Fatal(err)
		}
		ok, err = nd.Get(ctx, "key_test_del", &i)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatal("should be false")
		}
	})

}
