package etcdkv

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdStore struct {
	client *clientv3.Client
}

func NewStore(endpoints []string) (*EtcdStore, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}
	return &EtcdStore{client: cli}, nil
}

func (es *EtcdStore) Has(ctx context.Context, key string) (bool, error) {
	resp, err := es.client.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return len(resp.Kvs) > 0, nil
}

func (es *EtcdStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	resp, err := es.client.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}
	if len(resp.Kvs) == 0 {
		return nil, false, nil
	}
	return resp.Kvs[0].Value, true, nil
}

func (es *EtcdStore) Set(ctx context.Context, key string, val []byte) error {
	_, err := es.client.Put(ctx, key, string(val))
	return err
}

func (es *EtcdStore) Del(ctx context.Context, key string) (bool, error) {
	resp, err := es.client.Delete(ctx, key)
	if err != nil {
		return false, err
	}
	return resp.Deleted > 0, nil
}

func (es *EtcdStore) ForEach(ctx context.Context, fn func(ctx context.Context, key string, value []byte) bool) error {
	resp, err := es.client.Get(ctx, "", clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range resp.Kvs {
		if !fn(ctx, string(kv.Key), kv.Value) {
			break
		}
	}
	return nil
}
