package rediskv

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

var mr *miniredis.Miniredis

// TestMain 控制测试的setup和teardown
func TestMain(m *testing.M) {
	var err error
	mr, err = miniredis.Run()
	if err != nil {
		fmt.Printf("Could not start miniredis: %v", err)
		os.Exit(1)
	}
	defer mr.Close()

	// 执行测试
	code := m.Run()

	// Exit with the result code from the tests
	os.Exit(code)
}

// setup 使用 miniredis 创建一个 RedisStore
func setup() *RedisStore {
	options := &redis.Options{
		Addr: mr.Addr(),
	}
	prefix := fmt.Sprintf("test:prefix:%d", rand.Int())
	return NewStore(options, prefix)
}

func TestSetAndGet(t *testing.T) {
	store := setup()
	ctx := context.Background()

	key := "key1"
	value := []byte("value1")

	err := store.Set(ctx, key, value)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	got, ok, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("Expected key to exist")
	}
	if string(got) != string(value) {
		t.Fatalf("Expected %s, got %s", value, got)
	}
}

func TestHas(t *testing.T) {
	store := setup()
	ctx := context.Background()

	key := "key2"
	value := []byte("value2")

	_, err := store.Has(ctx, key)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = store.Set(ctx, key, value)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	has, err := store.Has(ctx, key)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !has {
		t.Fatalf("Expected key to exist")
	}
}

func TestDel(t *testing.T) {
	store := setup()
	ctx := context.Background()

	key := "key3"
	value := []byte("value3")

	err := store.Set(ctx, key, value)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ok, err := store.Del(ctx, key)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("Expected key to be deleted")
	}

	_, ok, err = store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if ok {
		t.Fatalf("Expected key not to exist")
	}
}

func TestForEach(t *testing.T) {
	store := setup()
	ctx := context.Background()

	keys := []string{"key4", "key5"}
	values := [][]byte{[]byte("value4"), []byte("value5")}

	for i, key := range keys {
		err := store.Set(ctx, key, values[i])
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	}

	err := store.ForEach(ctx, func(ctx context.Context, key string, value []byte) bool {
		found := false
		for i, k := range keys {
			if k == key && string(values[i]) == string(value) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected key-value pair: %s=%s", key, value)
		}
		return true
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
