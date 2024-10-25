package bboltkv

import (
	"context"
	"errors"
	"time"

	"go.etcd.io/bbolt"
)

const (
	DefaultBucketName = "_kvstore"
)

type BboltStore struct {
	db *bbolt.DB
}

func NewStore(dbPath string) (*BboltStore, error) {
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	return &BboltStore{db: db}, nil
}

func (bs *BboltStore) Has(ctx context.Context, key string) (bool, error) {
	var has bool
	err := bs.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(DefaultBucketName))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		val := bucket.Get([]byte(key))
		has = val != nil
		return nil
	})
	return has, err
}

func (bs *BboltStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var data []byte
	err := bs.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(DefaultBucketName))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		data = bucket.Get([]byte(key))
		if data == nil {
			return nil
		}
		data = append([]byte{}, data...) // Clone data for safety
		return nil
	})
	return data, data != nil, err
}

func (bs *BboltStore) Set(ctx context.Context, key string, val []byte) error {
	return bs.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(DefaultBucketName))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(key), val)
	})
}

func (bs *BboltStore) Del(ctx context.Context, key string) (bool, error) {
	var deleted bool
	err := bs.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(DefaultBucketName))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		val := bucket.Get([]byte(key))
		if val == nil {
			return nil
		}
		err := bucket.Delete([]byte(key))
		deleted = (err == nil)
		return err
	})
	return deleted, err
}

func (bs *BboltStore) ForEach(ctx context.Context, fn func(ctx context.Context, key string, value []byte) (ok bool)) error {
	return bs.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(DefaultBucketName))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		return bucket.ForEach(func(k, v []byte) error {
			if !fn(ctx, string(k), v) {
				return errors.New("iteration stopped")
			}
			return nil
		})
	})
}
