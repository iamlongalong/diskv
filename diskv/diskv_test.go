package diskv

import (
	"context"
	"fmt"
	"testing"
)

func TestDiskv(t *testing.T) {
	ctx := context.Background()
	dir := "./test"

	var db *Diskv
	var err error

	config := DefaultCreateConfig
	config.Dir = dir

	db, err = CreateDB(ctx, &config)
	// db, err := OpenDB(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("set key", func(t *testing.T) {
		err = db.Set(ctx, "key", []byte("value"))
		if err != nil {
			t.Fatal(err)
		}

		val, ok, err := db.Get(ctx, "key")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("key not found")
		}
		if string(val) != "value" {
			t.Fatal("value not match")
		}

		err = db.Set(ctx, "key", []byte("value2"))
		if err != nil {
			t.Fatal(err)
		}

		val, ok, err = db.Get(ctx, "key")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("key not found")
		}
		if string(val) != "value2" {
			t.Fatal("value not match")
		}

		err = db.SetString(ctx, "key2", "value2")
		if err != nil {
			t.Fatal(err)
		}
		strval, ok, err := db.GetString(ctx, "key2")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("key not found")
		}
		if strval != "value2" {
			t.Fatal("value not match")
		}

		// has key
		ok, err = db.Has(ctx, "key")
		if err != nil {
			t.Fatalf("has error: %v", err)
		}
		if !ok {
			t.Fatal("key not found")
		}

		ok, err = db.Has(ctx, "key2x")
		if err != nil {
			t.Fatalf("has error: %v", err)
		}

		if ok {
			t.Fatal("key found")
		}

		// del key
		has, err := db.Del(ctx, "key2x")
		if err != nil {
			t.Fatal(err)
		}

		if has {
			t.Fatal("key found")
		}

		has, err = db.Del(ctx, "key2")
		if err != nil {
			t.Fatal(err)
		}

		if !has {
			t.Fatal("key not found")
		}

	})

	t.Run("foreach", func(t *testing.T) {
		db.Set(ctx, "key3", []byte("value3"))
		db.Set(ctx, "key4", []byte("value4"))
		db.Set(ctx, "key5", []byte("value5"))
		db.Set(ctx, "key6", []byte("value6"))

		err := db.ForEach(ctx, func(ctx context.Context, key string, value []byte) (ok bool) {
			fmt.Println(key, string(value))
			return true
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("max len", func(t *testing.T) {
		key20 := "12345678901234567890"
		err := db.Set(ctx, key20, []byte(""))
		if err != nil {
			t.Fatalf("should not has error")
		}

		key30 := "123456789012345678901234567890"
		err = db.Set(ctx, key30, []byte(""))
		if err == nil {
			t.Fatalf("should get key too long error")
		}
	})

	t.Run("migrate", func(t *testing.T) {
		err := db.MigrateValue(ctx)
		if err != nil {
			t.Fatal(err)
		}

		err = db.MigrateIdx(ctx, &CreateConfig{
			Dir:     dir,
			KeysLen: 20,
			MaxLen:  64,
		})
		if err != nil {
			t.Fatal(err)
		}

		key30 := "123456789012345678901234567890"
		err = db.Set(ctx, key30, []byte("xxxxxxx"))
		if err != nil {
			t.Fatal(err)
		}
	})
}

func BenchmarkDiskv(b *testing.B) {
	ctx := context.Background()
	dir := "./test/bench"

	var db *Diskv
	var err error
	config := CreateConfig{
		KeysLen: 10000,
		MaxLen:  128,
	}
	config.Dir = dir

	db, err = CreateDB(ctx, &config)
	if err != nil {
		b.Fatal(err)
	}

	keynums := 5000
	for i := 0; i < b.N; i++ {
		name := i % keynums
		nameStr := fmt.Sprintf("%d", name)
		err = db.Set(ctx, nameStr, []byte("value: "+nameStr))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDecode(t *testing.T) {
	x := "_del[xx]"
	op, item, err := decodeValue("xx", []byte(x))
	if err != nil {
		t.Fatal(err)
	}

	if item.key != "xx" {
		t.Fatal("key not match")
	}

	if item.value != nil {
		t.Fatal("value not match")
	}

	if op != "_del" {
		t.Fatal("op not match")
	}

	fmt.Printf("op: [%s], key: [%s], val: [%s]\n", op, item.key, string(item.value))
}
