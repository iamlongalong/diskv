package bboltkv

import (
	"context"
	"os"
	"testing"
)

func TestBboltStore(t *testing.T) {
	// 创建一个临时数据库文件
	dbPath := "test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		key, value := "key1", []byte("value1")
		err := store.Set(ctx, key, value)
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}

		got, ok, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("failed to get value: %v", err)
		}

		if !ok || string(got) != string(value) {
			t.Errorf("got %v, want %v", got, value)
		}
	})

	t.Run("Has", func(t *testing.T) {
		key := "key1"
		has, err := store.Has(ctx, key)
		if err != nil || !has {
			t.Errorf("expected key %v to exist, got has %v with error %v", key, has, err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := "key1"
		ok, err := store.Del(ctx, key)
		if err != nil || !ok {
			t.Errorf("failed to delete key: %v, error: %v", key, err)
		}

		_, ok, err = store.Get(ctx, key)
		if err != nil {
			t.Fatalf("failed to get value: %v", err)
		}

		if ok {
			t.Errorf("key %v should not exist after deletion", key)
		}
	})

	t.Run("ForEach", func(t *testing.T) {
		_ = store.Set(ctx, "key2", []byte("value2"))
		_ = store.Set(ctx, "key3", []byte("value3"))

		expectedKeys := map[string]bool{"key2": true, "key3": true}
		err := store.ForEach(ctx, func(ctx context.Context, key string, value []byte) bool {
			if !expectedKeys[key] {
				t.Errorf("unexpected key %v", key)
			}
			delete(expectedKeys, key)
			return true
		})

		if err != nil {
			t.Fatalf("failed to iterate over keys: %v", err)
		}

		if len(expectedKeys) != 0 {
			t.Errorf("missing keys: %v", expectedKeys)
		}
	})
}
