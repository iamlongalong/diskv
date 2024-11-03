package dblock

import (
	"context"
	"io"
)

type DBlock struct {
}

func (db *DBlock) Get(ctx context.Context, index int) ([]byte, error) {
	return nil, nil
}

func (db *DBlock) Set(ctx context.Context, index int, value []byte) (idx int, err error) {
	return index, nil
}

func (db *DBlock) Del(ctx context.Context, index int) error {
	return nil
}

type ReaderWriterAt interface {
	io.ReaderAt
	io.WriterAt
}
