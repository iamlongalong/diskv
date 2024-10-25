package etcdkv

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/server/v3/embed"
)

func startEmbeddedEtcd(t *testing.T) (*embed.Etcd, []string) {
	cfg := embed.NewConfig()
	cfg.Dir = t.TempDir()
	e, err := embed.StartEtcd(cfg)
	if err != nil {
		t.Fatalf("Failed to start embedded etcd: %v", err)
	}
	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(10 * time.Second):
		t.Fatalf("Embedded etcd server took too long to start")
	}
	endpoints := []string{cfg.AdvertiseClientUrls[0].String()}
	return e, endpoints
}

func TestEtcdStore(t *testing.T) {
	e, endpoints := startEmbeddedEtcd(t)
	defer e.Close()

	store, err := NewStore(endpoints)
	if err != nil {
		t.Fatalf("Failed to create EtcdStore: %v", err)
	}
	defer store.client.Close()

	ctx := context.Background()
	key := "testKey"
	value := []byte("testValue")

	// Test Set
	err = store.Set(ctx, key, value)
	assert.NoError(t, err)

	// Test Has
	has, err := store.Has(ctx, key)
	assert.NoError(t, err)
	assert.True(t, has)

	// Test Get
	data, ok, err := store.Get(ctx, key)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, value, data)

	// Test Del
	ok, err = store.Del(ctx, key)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Test Has after delete
	has, err = store.Has(ctx, key)
	assert.NoError(t, err)
	assert.False(t, has)

	// Test ForEach
	err = store.Set(ctx, "key1", []byte("value1"))
	assert.NoError(t, err)
	err = store.Set(ctx, "key2", []byte("value2"))
	assert.NoError(t, err)

	keys := make(map[string][]byte)
	err = store.ForEach(ctx, func(ctx context.Context, k string, v []byte) bool {
		keys[k] = v
		return true
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(keys))
	assert.Equal(t, []byte("value1"), keys["key1"])
	assert.Equal(t, []byte("value2"), keys["key2"])
}
