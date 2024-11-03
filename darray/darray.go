package darray

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// const ( // darray with meta, may be a file
// 	metaSize = 32
// )

type Meta struct {
	Size   int `json:"size"`   // 每个元素的大小
	Length int `json:"length"` // 数组长度
}

type DArray struct {
	rw ReaderWriterAt

	meta Meta
}

func New(rw ReaderWriterAt, meta Meta) *DArray {
	return &DArray{
		rw:   rw,
		meta: meta,
	}
}

// 8[datalength][data]

var (
	ErrIdxOutOfRange = errors.New("index out of range")
)

func (da *DArray) Get(ctx context.Context, index int) ([]byte, error) {
	if index >= da.meta.Length || index < 0 {
		return nil, ErrIdxOutOfRange
	}

	from := index * da.meta.Size

	data := make([]byte, da.meta.Size)
	if _, err := da.rw.ReadAt(data, int64(from)); err != nil {
		return nil, err
	}

	lengthStr := data[:8]
	length := 0
	_, err := fmt.Sscanf(string(lengthStr), "%d", &length)
	if err != nil {
		return nil, fmt.Errorf("parse value meta error: %s", err)
	}

	if length == 0 {
		return nil, nil
	}

	return data[8 : length+8], nil
}

func (da *DArray) Set(ctx context.Context, index int, value []byte) error {
	if index >= da.meta.Length || index < 0 {
		return ErrIdxOutOfRange
	}

	if len(value) > da.meta.Size {
		return errors.New("value size not match")
	}

	// if value == nil {//}

	from := index * da.meta.Size
	value = append([]byte(fmt.Sprintf("%08d", len(value))), value...)

	_, err := da.rw.WriteAt(value, int64(from))
	if err != nil {
		return err
	}

	return nil
}

type ReaderWriterAt interface {
	io.ReaderAt
	io.WriterAt
}
