package sqlitekv

import (
	"context"
	"os"
	"testing"
)

func TestSqliteStore(t *testing.T) {
	// 临时数据库文件
	dbPath := "test.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	ctx := context.Background()

	// Test Set and Get
	key := "foo"
	value := []byte("bar")
	err = store.Set(ctx, key, value)
	if err != nil {
		t.Errorf("Failed to set value: %v", err)
	}
	got, ok, err := store.Get(ctx, key)
	if err != nil || !ok {
		t.Errorf("Failed to get value: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("Expected %s, got %s", value, got)
	}

	// Test Has
	has, err := store.Has(ctx, key)
	if err != nil || !has {
		t.Errorf("Expected key to exist: %v", err)
	}

	// Test Del
	ok, err = store.Del(ctx, key)
	if err != nil || !ok {
		t.Errorf("Failed to delete key: %v", err)
	}
	_, ok, err = store.Get(ctx, key)
	if ok {
		t.Errorf("Expected key to be deleted")
	}

	// Test ForEach
	keys := []string{"key1", "key2", "key3"}
	values := [][]byte{[]byte("val1"), []byte("val2"), []byte("val3")}
	for i, k := range keys {
		err = store.Set(ctx, k, values[i])
		if err != nil {
			t.Fatalf("Failed to set key %s: %v", k, err)
		}
	}

	count := 0
	err = store.ForEach(ctx, func(ctx context.Context, key string, value []byte) (ok bool) {
		count++
		return true
	})
	if err != nil {
		t.Errorf("Error during iteration: %v", err)
	}
	if count != len(keys) {
		t.Errorf("Expected %d items, got %d", len(keys), count)
	}
}
