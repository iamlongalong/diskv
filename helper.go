package diskv

import (
	"context"
	"fmt"
	"os"
)

// 迁移 idx 文件
func (d *Diskv) MigrateIdx(ctx context.Context, toConfig *CreateConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	toConfig.Dir = d.dir
	toIdxFile := d.idxFileName(d.dir) + ".tmp"

	toIdx, err := d.createIdx(ctx, toIdxFile, *toConfig)
	if err != nil {
		return fmt.Errorf("create idx file error: %s", err)
	}

	ferr := d.forEachKey(ctx, func(ctx context.Context, valMeta *valueMeta) (ok bool) {
		err = toIdx.setValueMeta(ctx, valMeta)
		if err != nil {
			return false
		}

		return true
	})
	if ferr != nil {
		return fmt.Errorf("forEachKey error: %s", ferr)
	}

	if err != nil {
		return fmt.Errorf("forEachKey error in func: %s", err)
	}

	err = migrateFile(ctx, toIdxFile, d.idxFile, false)
	if err != nil {
		return fmt.Errorf("migrate file error: %s", err)
	}

	err = d.openDB(ctx, d.dir)
	if err != nil {
		return fmt.Errorf("reopen db file error: %s", err)
	}

	return nil
}

// 迁移 value 文件，用于把 del 等操作去除掉
func (d *Diskv) MigrateValue(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	toValueFile := d.dbFileName(d.dir) + ".tmp"

	dbstore, err := d.getOrCreateDBStore(toValueFile)
	if err != nil {
		return fmt.Errorf("create db file error: %s", err)
	}

	idxMeta, err := d.idx.getIdxMeta(ctx)
	if err != nil {
		return fmt.Errorf("get idx meta error: %s", err)
	}

	toValueIdxFile := d.idxFileName(d.dir) + ".tmp"
	nidx, err := d.createIdx(ctx, toValueIdxFile, CreateConfig{
		Dir: d.dir,
		// KeySize:      idxMeta.keySize,
		KeysLen: idxMeta.keysLen,
		// OffsetSize:   idxMeta.offsetSize,
		// ValueLenSize: idxMeta.valueLenSize,
		MaxLen: idxMeta.maxLength,
	})
	if err != nil {
		return fmt.Errorf("create idx file error: %s", err)
	}

	ferr := d.forEach(ctx, func(ctx context.Context, key string, value []byte) (ok bool) {
		var valueMeta *valueMeta
		valueMeta, err = dbstore.write(ctx, &valueItem{key: key, value: value})
		if err != nil {
			return false
		}

		err = nidx.setValueMeta(ctx, valueMeta)
		if err != nil {
			return false
		}

		return true
	})
	if ferr != nil {
		return fmt.Errorf("forEachKey error: %s", ferr)
	}

	if err != nil {
		return fmt.Errorf("forEachKey error in func: %s", err)
	}

	err = migrateFile(ctx, toValueFile, d.dbFile, false)
	if err != nil {
		return fmt.Errorf("migrate file error: %s", err)
	}

	err = migrateFile(ctx, toValueIdxFile, d.idxFile, false)
	if err != nil {
		return fmt.Errorf("migrate file error: %s", err)
	}

	err = d.openDB(ctx, d.dir)
	if err != nil {
		return fmt.Errorf("reopen db file error: %s", err)
	}
	return nil
}

func migrateFile(ctx context.Context, from string, to string, removeBak bool) error {
	toBackFile := to + "._bak"
	err := os.RemoveAll(toBackFile)
	if err != nil {
		return fmt.Errorf("remove old bak file [%s] error: %s", toBackFile, err)
	}

	err = os.Rename(to, toBackFile)
	if err != nil {
		return fmt.Errorf("rename old file [%s => %s] error: %s", to, toBackFile, err)
	}

	err = os.Rename(from, to)
	if err != nil {
		return fmt.Errorf("rename new file [%s => %s] error: %s", from, to, err)
	}

	if removeBak {
		err = os.RemoveAll(toBackFile)
		if err != nil {
			return fmt.Errorf("remove old bak file [%s] error: %s", toBackFile, err)
		}
	}
	return nil
}
